package cli

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/Wather17/higher/internal/storage"
	"github.com/Wather17/higher/internal/subscription"
	"github.com/spf13/cobra"
)

func newSubscriptionCommand(dependencies Dependencies) *cobra.Command {
	command := &cobra.Command{
		Use:   "subscription",
		Short: "Gerencie suas assinaturas",
		Args:  cobra.NoArgs,
	}
	command.AddCommand(
		newSubscriptionAddCommand(dependencies),
		newSubscriptionListCommand(dependencies),
		newSubscriptionEditCommand(dependencies),
		newSubscriptionCancelCommand(dependencies),
		newSubscriptionReactivateCommand(dependencies),
	)
	return command
}

func newSubscriptionAddCommand(dependencies Dependencies) *cobra.Command {
	var name string
	var amount string
	var currency string
	var period string
	var nextCharge string
	var note string

	command := &cobra.Command{
		Use:   "add",
		Short: "Adicione uma assinatura",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			amountCents, err := subscription.ParseAmount(amount)
			if err != nil {
				return err
			}
			parsedCurrency, err := subscription.ParseCurrency(currency)
			if err != nil {
				return err
			}
			parsedPeriod, err := subscription.ParsePeriod(period)
			if err != nil {
				return err
			}

			store, err := dependencies.store()
			if err != nil {
				return err
			}
			document, err := store.Load()
			if err != nil {
				return err
			}
			item, err := document.Add(subscription.AddInput{
				Name:        name,
				AmountCents: amountCents,
				Currency:    parsedCurrency,
				Period:      parsedPeriod,
				NextCharge:  nextCharge,
				Note:        note,
			})
			if err != nil {
				return err
			}
			if err := store.Save(document); err != nil {
				return err
			}

			fmt.Fprintf(command.OutOrStdout(), "Assinatura #%d criada.\n", item.ID)
			return nil
		},
	}
	command.Flags().StringVar(&name, "name", "", "nome da assinatura")
	command.Flags().StringVar(&amount, "amount", "", "valor usando ponto decimal, por exemplo 55.90")
	command.Flags().StringVar(&currency, "currency", string(subscription.CurrencyBRL), "moeda: BRL ou USD")
	command.Flags().StringVar(&period, "period", "", "periodicidade: monthly ou yearly")
	command.Flags().StringVar(&nextCharge, "next-charge", "", "próxima cobrança no formato YYYY-MM-DD")
	command.Flags().StringVar(&note, "note", "", "observação opcional")
	_ = command.MarkFlagRequired("name")
	_ = command.MarkFlagRequired("amount")
	_ = command.MarkFlagRequired("period")
	return command
}

func newSubscriptionListCommand(dependencies Dependencies) *cobra.Command {
	var all bool

	command := &cobra.Command{
		Use:   "list",
		Short: "Liste suas assinaturas",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			store, err := dependencies.store()
			if err != nil {
				return err
			}
			document, err := store.Load()
			if err != nil {
				return err
			}

			items := make([]subscription.Subscription, 0, len(document.Subscriptions))
			for _, item := range document.Subscriptions {
				if all || item.Status == subscription.StatusActive {
					items = append(items, item)
				}
			}
			if err := writeSubscriptionList(command.OutOrStdout(), items); err != nil {
				return err
			}
			return writeTotals(command.OutOrStdout(), document.Subscriptions)
		},
	}
	command.Flags().BoolVar(&all, "all", false, "inclui assinaturas canceladas")
	return command
}

func newSubscriptionEditCommand(dependencies Dependencies) *cobra.Command {
	var id int
	var name string
	var amount string
	var currency string
	var period string
	var nextCharge string
	var clearNextCharge bool
	var note string
	var clearNote bool

	command := &cobra.Command{
		Use:   "edit",
		Short: "Edite uma assinatura",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			flags := command.Flags()
			if flags.Changed("next-charge") && clearNextCharge || flags.Changed("note") && clearNote {
				return subscription.ErrConflictingFields
			}

			input := subscription.EditInput{}
			if flags.Changed("name") {
				input.Name = &name
			}
			if flags.Changed("amount") {
				amountCents, err := subscription.ParseAmount(amount)
				if err != nil {
					return err
				}
				input.AmountCents = &amountCents
			}
			if flags.Changed("currency") {
				parsedCurrency, err := subscription.ParseCurrency(currency)
				if err != nil {
					return err
				}
				input.Currency = &parsedCurrency
			}
			if flags.Changed("period") {
				parsedPeriod, err := subscription.ParsePeriod(period)
				if err != nil {
					return err
				}
				input.Period = &parsedPeriod
			}
			if flags.Changed("next-charge") {
				input.NextCharge = &nextCharge
			}
			input.ClearNextCharge = clearNextCharge
			if flags.Changed("note") {
				input.Note = &note
			}
			input.ClearNote = clearNote

			store, err := dependencies.store()
			if err != nil {
				return err
			}
			document, err := store.Load()
			if err != nil {
				return err
			}
			item, err := document.Edit(id, input, dependencies.Now())
			if err != nil {
				return err
			}
			if err := store.Save(document); err != nil {
				return err
			}

			fmt.Fprintf(command.OutOrStdout(), "Assinatura #%d atualizada.\n", item.ID)
			return nil
		},
	}
	command.Flags().IntVar(&id, "id", 0, "ID da assinatura")
	command.Flags().StringVar(&name, "name", "", "novo nome")
	command.Flags().StringVar(&amount, "amount", "", "novo valor usando ponto decimal")
	command.Flags().StringVar(&currency, "currency", "", "nova moeda: BRL ou USD")
	command.Flags().StringVar(&period, "period", "", "nova periodicidade: monthly ou yearly")
	command.Flags().StringVar(&nextCharge, "next-charge", "", "nova próxima cobrança no formato YYYY-MM-DD")
	command.Flags().BoolVar(&clearNextCharge, "clear-next-charge", false, "remover a próxima cobrança")
	command.Flags().StringVar(&note, "note", "", "nova observação")
	command.Flags().BoolVar(&clearNote, "clear-note", false, "remover a observação")
	_ = command.MarkFlagRequired("id")
	return command
}

func newSubscriptionCancelCommand(dependencies Dependencies) *cobra.Command {
	var id int
	command := &cobra.Command{
		Use:   "cancel",
		Short: "Cancele uma assinatura sem apagar seu registro",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			store, err := dependencies.store()
			if err != nil {
				return err
			}
			document, err := store.Load()
			if err != nil {
				return err
			}
			item, err := document.Cancel(id, dependencies.Now())
			if err != nil {
				return err
			}
			if err := store.Save(document); err != nil {
				return err
			}

			fmt.Fprintf(command.OutOrStdout(), "Assinatura #%d cancelada.\n", item.ID)
			return nil
		},
	}
	command.Flags().IntVar(&id, "id", 0, "ID da assinatura")
	_ = command.MarkFlagRequired("id")
	return command
}

func newSubscriptionReactivateCommand(dependencies Dependencies) *cobra.Command {
	var id int
	command := &cobra.Command{
		Use:   "reactivate",
		Short: "Reative uma assinatura cancelada",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			store, err := dependencies.store()
			if err != nil {
				return err
			}
			document, err := store.Load()
			if err != nil {
				return err
			}
			item, err := document.Reactivate(id, dependencies.Now())
			if err != nil {
				return err
			}
			if err := store.Save(document); err != nil {
				return err
			}

			fmt.Fprintf(command.OutOrStdout(), "Assinatura #%d reativada.\n", item.ID)
			return nil
		},
	}
	command.Flags().IntVar(&id, "id", 0, "ID da assinatura")
	_ = command.MarkFlagRequired("id")
	return command
}

func writeSubscriptionList(output io.Writer, items []subscription.Subscription) error {
	if len(items) == 0 {
		_, err := fmt.Fprintln(output, "Nenhuma assinatura encontrada.")
		return err
	}

	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "ID\tNOME\tVALOR\tPERIODICIDADE\tPRÓXIMA COBRANÇA\tSTATUS"); err != nil {
		return err
	}
	for _, item := range items {
		nextCharge := item.NextCharge
		if nextCharge == "" {
			nextCharge = "-"
		}
		if _, err := fmt.Fprintf(writer, "%d\t%s\t%s %s\t%s\t%s\t%s\n", item.ID, item.Name, item.Currency, subscription.FormatAmount(item.AmountCents), item.Period, nextCharge, item.Status); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func writeTotals(output io.Writer, items []subscription.Subscription) error {
	totals, err := subscription.CalculateTotals(items)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(output, "\nTotais mensais (ativas):"); err != nil {
		return err
	}
	if err := writeCurrencyTotals(output, totals, true); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(output, "Totais anuais (ativas):"); err != nil {
		return err
	}
	return writeCurrencyTotals(output, totals, false)
}

func writeCurrencyTotals(output io.Writer, totals map[subscription.Currency]subscription.Totals, monthly bool) error {
	written := false
	for _, currency := range subscription.SupportedCurrencies {
		current, ok := totals[currency]
		if !ok {
			continue
		}
		amount := current.AnnualCents
		if monthly {
			amount = current.MonthlyCents
		}
		if _, err := fmt.Fprintf(output, "%s %s\n", currency, subscription.FormatAmount(amount)); err != nil {
			return err
		}
		written = true
	}
	if !written {
		_, err := fmt.Fprintln(output, "Nenhuma assinatura ativa.")
		return err
	}
	return nil
}

func (dependencies Dependencies) store() (*storage.Store, error) {
	if dependencies.Store != nil {
		return dependencies.Store, nil
	}
	return storage.NewDefaultStore()
}
