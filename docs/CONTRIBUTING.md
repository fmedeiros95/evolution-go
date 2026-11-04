# Manutenção dos contratos da API

O contrato executado é definido por `pkg/routes/routes.go`, pelos handlers, estruturas de requisição e middlewares. A documentação precisa acompanhar essas fontes.

## Arquivos

- Anotações `@Router`, `@Param`, `@Security`, `@Success` e `@Failure`: junto aos handlers.
- Estruturas Go: nomes JSON, tipos, exemplos e campos obrigatórios. As tags `validate:"required"` descrevem as verificações já feitas pelos handlers; não adicionam validação ao binding do Gin.
- `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`: gerados, sem edição manual.
- `.swaggo`: mapeia tipos com serialização própria. Por exemplo, JIDs são strings JSON, apesar da estrutura interna da biblioteca.
- `Evolution GO.postman_collection.json`: exemplos para todas as operações HTTP. Use `adminToken` nas rotas administrativas e `token` nas rotas de instância.
- `docs/wiki`: explicações de fluxos, exemplos e formatos alternativos.

## Atualizar

Inicialize o submódulo antes de gerar os contratos:

```sh
git submodule update --init --recursive
go run github.com/swaggo/swag/cmd/swag@v1.16.3 init -g cmd/evolution-go/main.go -o ./docs --parseDependency --parseInternal --propertyStrategy pascalcase
go run ./tools/check-api-docs
go test ./tools/check-api-docs
```

`make swagger` e `make docs-check` executam os mesmos comandos. A versão do gerador é fixa para evitar diferenças causadas por instalações locais. `--propertyStrategy pascalcase` preserva o nome dos campos Go sem tag JSON, como `Info`, `Message`, `Chat` e `ID`; campos com tag explícita mantêm o nome da tag.

O CI compara as rotas registradas com Swagger e Postman, confere os esquemas de autenticação e respostas 401 das rotas protegidas e regenera os arquivos para detectar saídas desatualizadas. Essa checagem não confirma respostas de uma sessão WhatsApp real nem valida automaticamente todos os exemplos em prosa.

## Autenticação

O header é `apikey` nos dois casos:

- `AdminApiKey`: valor de `GLOBAL_API_KEY`, para as rotas administrativas de instância.
- `InstanceApiKey`: token da instância, para as demais operações protegidas.

Chave ausente ou inválida retorna 401. `/server/ok` é público. O fork não exige ativação externa e não oferece rotas `/license/*`. Manager, arquivos estáticos, Swagger e o upgrade WebSocket são descritos nos guias, fora da comparação de operações REST.

## Uploads de mídia

Swagger 2.0 não permite declarar, na mesma operação, um body JSON e parâmetros `formData`. Nas operações que aceitam os dois formatos, a interface Swagger apresenta o JSON; as descrições e o Postman explicam o multipart.

`POST /send/media` aceita:

- JSON: `number`, `type` e `url` obrigatórios. `url` aceita endereço HTTP(S) ou base64 sem prefixo `data:`.
- Multipart: `number`, `type` e `file` obrigatórios. Campos opcionais: `caption`, `filename`, `id`, `delay`, `mentionAll`, `mentionedJid` repetido, `quoted.messageId` e `quoted.participant`.

Exemplo multipart:

```sh
curl -X POST http://localhost:8080/send/media \
  -H "apikey: TOKEN_DA_INSTANCIA" \
  -F "number=5511999999999" \
  -F "type=document" \
  -F "file=@arquivo.pdf" \
  -F "mentionedJid=5511888888888@s.whatsapp.net" \
  -F "quoted.messageId=ID_DA_MENSAGEM" \
  -F "quoted.participant=5511999999999@s.whatsapp.net"
```

`POST /send/status/media` usa JSON com `type` e `url`, ou multipart com `type` e `file`. Nesse endpoint `type` deve ser `image` ou `video`.

## Mensagens editadas recebidas

Uma edição criptografada do tipo `MESSAGE_EDIT` mantém os dados originais no webhook `Message` e acrescenta o conteúdo descriptografado em `data.Message.decryptedEdit`. A identificação da mensagem original permanece em `data.Message.secretEncryptedMessage.targetMessageKey`. Se a descriptografia falhar, o campo adicional não é enviado. Edições tradicionais continuam nos formatos já entregues pela biblioteca.
