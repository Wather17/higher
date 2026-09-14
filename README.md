# Higher

Higher é um CLI pessoal de gestão financeira, escrito em Go. A primeira funcionalidade acompanha assinaturas recorrentes armazenadas localmente em JSON.

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
