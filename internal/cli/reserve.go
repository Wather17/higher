package cli

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/Wather17/higher/internal/reserve"
	"github.com/Wather17/higher/internal/storage"
	"github.com/Wather17/higher/internal/subscription"
	"github.com/spf13/cobra"
)

func newReserveCommand(dependencies Dependencies) *cobra.Command {
	command := &cobra.Command{
		Use:   "reserve",
		Short: "Gerencie sua reserva de emergência",
		Args:  cobra.NoArgs,
	}
	command.AddCommand(
		newReserveSetupCommand(dependencies),
		newReserveDepositCommand(dependencies),
		newReserveWithdrawCommand(dependencies),
		newReserveStatusCommand(dependencies),
		newReserveListCommand(dependencies),
		newReserveEditCommand(dependencies),
	)
	return command
}

func newReserveSetupCommand(dependencies Dependencies) *cobra.Command {
	var income string
	var saveRate string
	var target string

	command := &cobra.Command{
		Use:   "setup",
		Short: "Configure sua reserva de emergência",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			incomeCents, err := subscription.ParseAmount(income)
			if err != nil {
				return reserve.ErrInvalidIncome
			}
			rate, err := reserve.ParseRate(saveRate)
			if err != nil {
				return err
			}

			var targetCents *int64
			if command.Flags().Changed("target") {
				parsedTarget, err := subscription.ParseAmount(target)
				if err != nil {
					return reserve.ErrInvalidTarget
				}
				targetCents = &parsedTarget
			}

			store, err := dependencies.reserveStore()
			if err != nil {
				return err
			}
			document, err := store.Load()
			if err != nil {
				return err
			}
			if err := document.Setup(reserve.SetupInput{
				IncomeCents:         incomeCents,
				SaveRateBasisPoints: rate,
				TargetCents:         targetCents,
			}); err != nil {
				return err
			}
			if err := store.Save(document); err != nil {
				return err
			}

			fmt.Fprintln(command.OutOrStdout(), "Reserva de emergência criada.")
			return nil
		},
	}
	command.Flags().StringVar(&income, "income", "", "renda líquida mensal, por exemplo 5000.00")
	command.Flags().StringVar(&saveRate, "save-rate", "", "percentual mensal para guardar, por exemplo 10 ou 10.50")
	command.Flags().StringVar(&target, "target", "", "meta da reserva, por exemplo 30000.00; padrão: seis rendas")
	_ = command.MarkFlagRequired("income")
	_ = command.MarkFlagRequired("save-rate")
	return command
}

func newReserveDepositCommand(dependencies Dependencies) *cobra.Command {
	return newReserveEntryCommand(dependencies, reserve.EntryDeposit)
}

func newReserveWithdrawCommand(dependencies Dependencies) *cobra.Command {
	return newReserveEntryCommand(dependencies, reserve.EntryWithdrawal)
}

func newReserveEntryCommand(dependencies Dependencies, kind reserve.EntryKind) *cobra.Command {
	var amount string
	var date string
	var note string

	use := "deposit"
	short := "Registre um depósito na reserva"
	if kind == reserve.EntryWithdrawal {
		use = "withdraw"
		short = "Registre uma retirada da reserva"
	}

	command := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			amountCents, err := subscription.ParseAmount(amount)
			if err != nil {
				return err
			}

			store, err := dependencies.reserveStore()
			if err != nil {
				return err
			}
			document, err := store.Load()
			if err != nil {
				return err
			}
			input := reserve.EntryInput{AmountCents: amountCents, Date: date, Note: note}
			var entry reserve.Entry
			if kind == reserve.EntryDeposit {
				entry, err = document.Deposit(input, dependencies.Now())
			} else {
				entry, err = document.Withdraw(input, dependencies.Now())
			}
			if err != nil {
				return err
			}
			if err := store.Save(document); err != nil {
				return err
			}

			if kind == reserve.EntryDeposit {
				fmt.Fprintf(command.OutOrStdout(), "Depósito #%d registrado.\n", entry.ID)
			} else {
				fmt.Fprintf(command.OutOrStdout(), "Retirada #%d registrada.\n", entry.ID)
			}
			return nil
		},
	}
	command.Flags().StringVar(&amount, "amount", "", "valor usando ponto decimal, por exemplo 500.00")
	command.Flags().StringVar(&date, "date", "", "data no formato YYYY-MM-DD; padrão: hoje")
	command.Flags().StringVar(&note, "note", "", "observação opcional")
	_ = command.MarkFlagRequired("amount")
	return command
}

func newReserveStatusCommand(dependencies Dependencies) *cobra.Command {
	command := &cobra.Command{
		Use:   "status",
		Short: "Mostre o progresso da reserva",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			store, err := dependencies.reserveStore()
			if err != nil {
				return err
			}
			document, err := store.Load()
			if err != nil {
				return err
			}
			summary, err := document.Summary()
			if err != nil {
				return err
			}
			return writeReserveStatus(command.OutOrStdout(), summary)
		},
	}
	return command
}

func newReserveListCommand(dependencies Dependencies) *cobra.Command {
	command := &cobra.Command{
		Use:   "list",
		Short: "Liste os lançamentos da reserva",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			store, err := dependencies.reserveStore()
			if err != nil {
				return err
			}
			document, err := store.Load()
			if err != nil {
				return err
			}
			if !document.IsConfigured() {
				return reserve.ErrNotConfigured
			}
			return writeReserveList(command.OutOrStdout(), document.SortedEntries())
		},
	}
	return command
}

func newReserveEditCommand(dependencies Dependencies) *cobra.Command {
	var income string
	var saveRate string
	var target string

	command := &cobra.Command{
		Use:   "edit",
		Short: "Edite a configuração da reserva",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			flags := command.Flags()
			input := reserve.EditInput{}
			if flags.Changed("income") {
				incomeCents, err := subscription.ParseAmount(income)
				if err != nil {
					return reserve.ErrInvalidIncome
				}
				input.IncomeCents = &incomeCents
			}
			if flags.Changed("save-rate") {
				rate, err := reserve.ParseRate(saveRate)
				if err != nil {
					return err
				}
				input.SaveRateBasisPoints = &rate
			}
			if flags.Changed("target") {
				targetCents, err := subscription.ParseAmount(target)
				if err != nil {
					return reserve.ErrInvalidTarget
				}
				input.TargetCents = &targetCents
			}

			store, err := dependencies.reserveStore()
			if err != nil {
				return err
			}
			document, err := store.Load()
			if err != nil {
				return err
			}
			if err := document.Edit(input); err != nil {
				return err
			}
			if err := store.Save(document); err != nil {
				return err
			}

			fmt.Fprintln(command.OutOrStdout(), "Configuração da reserva atualizada.")
			return nil
		},
	}
	command.Flags().StringVar(&income, "income", "", "nova renda líquida mensal")
	command.Flags().StringVar(&saveRate, "save-rate", "", "novo percentual mensal para guardar")
	command.Flags().StringVar(&target, "target", "", "nova meta da reserva")
	return command
}

func writeReserveStatus(output io.Writer, summary reserve.Summary) error {
	if _, err := fmt.Fprintf(output, "Renda líquida mensal: BRL %s\n", subscription.FormatAmount(summary.IncomeCents)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "Percentual para guardar: %s\n", reserve.FormatRate(summary.SaveRateBasisPoints)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "Aporte mensal planejado: BRL %s\n", subscription.FormatAmount(summary.MonthlyContribution)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "Meta: BRL %s\n", subscription.FormatAmount(summary.TargetCents)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "Saldo: BRL %s\n", subscription.FormatAmount(summary.BalanceCents)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "Restante: BRL %s\n", subscription.FormatAmount(summary.RemainingCents)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "Progresso: %s\n", reserve.FormatRate(summary.ProgressBasisPoints)); err != nil {
		return err
	}
	if summary.Achieved {
		_, err := fmt.Fprintln(output, "Meta atingida.")
		return err
	}
	_, err := fmt.Fprintf(output, "Previsão: %d meses\n", summary.MonthsRemaining)
	return err
}

func writeReserveList(output io.Writer, entries []reserve.Entry) error {
	if len(entries) == 0 {
		_, err := fmt.Fprintln(output, "Nenhuma movimentação encontrada.")
		return err
	}

	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "ID\tTIPO\tVALOR\tDATA\tNOTA"); err != nil {
		return err
	}
	for _, entry := range entries {
		kind := "Depósito"
		if entry.Kind == reserve.EntryWithdrawal {
			kind = "Retirada"
		}
		note := entry.Note
		if note == "" {
			note = "-"
		}
		if _, err := fmt.Fprintf(writer, "%d\t%s\tBRL %s\t%s\t%s\n", entry.ID, kind, subscription.FormatAmount(entry.AmountCents), entry.Date, note); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func (dependencies Dependencies) reserveStore() (*storage.ReserveStore, error) {
	if dependencies.ReserveStore != nil {
		return dependencies.ReserveStore, nil
	}
	return storage.NewDefaultReserveStore()
}
