# Higher

Higher é um CLI pessoal de gestão financeira, escrito em Go. Ele acompanha assinaturas recorrentes e uma reserva de emergência armazenadas localmente em JSON.

## Desenvolvimento

```bash
go test ./...
go vet ./...
go build ./...
```

## Assinaturas

Os valores usam ponto decimal e a moeda padrão é BRL. O CLI também suporta USD, mantendo os totais separados.

```bash
go run ./cmd/higher subscription add \
  --name "Netflix" \
  --amount 55.90 \
  --period monthly \
  --next-charge 2026-10-01

go run ./cmd/higher subscription list
go run ./cmd/higher subscription edit --id 1 --amount 59.90
go run ./cmd/higher subscription cancel --id 1
go run ./cmd/higher subscription reactivate --id 1
```

`subscription list` mostra somente assinaturas ativas. Use `--all` para incluir canceladas. Os dados ficam no diretório de dados do usuário, com suporte a XDG no Linux.

## Reserva de emergência

A reserva usa renda líquida mensal e um percentual de economia. Sem `--target`, a meta inicial é de seis rendas mensais. Os valores usam BRL e ficam em um arquivo separado das assinaturas.

```bash
go run ./cmd/higher reserve setup \
  --income 5000.00 \
  --save-rate 10

go run ./cmd/higher reserve deposit --amount 500.00 --note "aporte mensal"
go run ./cmd/higher reserve withdraw --amount 200.00 --note "emergência médica"
go run ./cmd/higher reserve status
go run ./cmd/higher reserve list
go run ./cmd/higher reserve edit --save-rate 15
```

Use `--target` no `reserve setup` ou `reserve edit` para definir uma meta própria. Depósitos e retiradas aceitam `--date YYYY-MM-DD`; quando omitida, a data atual é usada. Retiradas maiores que o saldo são rejeitadas.
