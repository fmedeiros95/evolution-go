# Sistema de eventos

Eventos podem ser entregues por webhook, RabbitMQ, NATS e WebSocket. Os contratos abaixo correspondem aos emissores de `pkg/whatsmeow/service/whatsmeow.go` e `pkg/events`.

## Inscrições e tipos de eventos

`subscribe` usa categorias em maiúsculas. O campo `event` entregue ao consumidor usa o nome concreto do evento, com maiúsculas e minúsculas preservadas.

| Categoria de inscrição | Exemplos de eventos entregues |
|---|---|
| `MESSAGE` | `Message` |
| `SEND_MESSAGE` | `SendMessage` |
| `READ_RECEIPT` | `Receipt` |
| `PRESENCE` | `Presence` |
| `HISTORY_SYNC` | `HistorySync` |
| `CHAT_PRESENCE` | `ChatPresence`, `Archive` |
| `CALL` | `CallOffer`, `CallAccept`, `CallTerminate` |
| `CONNECTION` | `Connected`, `Disconnected`, `LoggedOut` |
| `QRCODE` | `QRCode` |
| `LABEL` | `LabelEdit`, `LabelAssociationChat`, `LabelAssociationMessage` |
| `CONTACT` | `Contact`, `PushName` |
| `GROUP` | `GroupInfo`, `JoinedGroup` |
| `NEWSLETTER` | `NewsletterJoin`, `NewsletterLeave` |
| `BUTTON_CLICK` | `ButtonClick` |
| `ALL` | Todos os eventos encaminhados pelo serviço |

A lista de categorias válidas está em `pkg/internal/event_types/event_types.go`. As categorias `GROUP` e `NEWSLETTER` também podem receber `Message` e `SendMessage` quando o chat termina em `@g.us` ou `@newsletter`. Portanto, o consumidor deve verificar `event` e os dados do chat.

## Configuração por instância

```sh
curl -X POST http://localhost:8080/instance/connect \
  -H "Content-Type: application/json" \
  -H "apikey: TOKEN_DA_INSTANCIA" \
  -d '{
    "webhookUrl": "https://meu-servidor.example/webhook",
    "subscribe": ["MESSAGE", "READ_RECEIPT", "BUTTON_CLICK"],
    "rabbitmqEnable": "disabled",
    "natsEnable": "disabled",
    "websocketEnable": "enabled"
  }'
```

Os nomes JSON são sensíveis à grafia: `webhookUrl`, `rabbitmqEnable`, `natsEnable` e `websocketEnable`. Os campos de transporte são strings. Para habilitar o encaminhamento, prefira `"enabled"`.

## Webhook

O emissor envia HTTP POST com `Content-Type: application/json`. Exemplo reduzido de mensagem recebida:

```json
{
  "event": "Message",
  "instanceId": "uuid-da-instancia",
  "instanceName": "vendas",
  "instanceToken": "TOKEN_DA_INSTANCIA",
  "data": {
    "Info": {
      "ID": "ID_DA_MENSAGEM",
      "Chat": "5511999999999@s.whatsapp.net",
      "Sender": "5511999999999@s.whatsapp.net",
      "IsFromMe": false,
      "Timestamp": "2026-11-04T12:00:00Z"
    },
    "Message": {
      "conversation": "Olá!"
    }
  }
}
```

`data.Info` e `data.Message` começam com maiúscula. Os campos internos da mensagem, como `conversation` e `extendedTextMessage`, usam os nomes JSON do protobuf. O exemplo omite campos que dependem do evento; não trate `Message` como um único formato de texto.

O envelope inclui `instanceToken`, que é uma credencial. Evite registrar o payload completo em logs compartilhados.

### Mensagens editadas

Para uma edição criptografada do tipo `MESSAGE_EDIT`, o serviço tenta descriptografar o conteúdo e acrescenta `data.Message.decryptedEdit`. O identificador da mensagem original permanece em `data.Message.secretEncryptedMessage.targetMessageKey`.

Se a descriptografia falhar, `decryptedEdit` não é enviado. Edições tradicionais continuam nos formatos entregues pela biblioteca, incluindo `protocolMessage` e mensagens com `IsEdit`.

### Exemplo de receptor

```javascript
import express from 'express';

const app = express();
app.use(express.json());

app.post('/webhook', (req, res) => {
  const { event, instanceId, data } = req.body;
  if (event === 'Message') {
    const message = data?.Message?.decryptedEdit ?? data?.Message;
    const text = message?.conversation ?? message?.extendedTextMessage?.text;
    console.log({ instanceId, messageId: data?.Info?.ID, text });
  }
  res.sendStatus(200);
});
app.listen(3000);
```

### Entrega e tentativas

O produtor considera qualquer resposta HTTP 2xx como sucesso. Em caso de erro, faz até cinco tentativas no total, com espera de 30 segundos após falhas. Não há garantia de entrega única: faça deduplicação considerando instância, tipo do evento e identificadores da mensagem.

`WEBHOOK_URL` configura o destino global. O produtor tenta esse destino e o `webhookUrl` da instância quando é acionado; não envia apenas para um deles. O encaminhamento também depende das inscrições e da configuração da instância. Se a mesma URL for configurada nos dois destinos, pode receber duas entregas.

## RabbitMQ

Configure a conexão com `AMQP_URL` e habilite a instância com `rabbitmqEnable: "enabled"`. O nome da fila por instância é o ID da instância seguido do nome concreto do evento, em minúsculas: `<instanceId>.message`, por exemplo.

Para filas globais, use `AMQP_GLOBAL_ENABLED=true` e configure `AMQP_GLOBAL_EVENTS` com as categorias desejadas. O envelope JSON é o mesmo encaminhado pelo serviço de eventos.

## NATS

Configure `NATS_URL` e `natsEnable: "enabled"`. Quando `NATS_URL` está vazio, a conexão NATS é desabilitada. O subject por instância segue `<instanceId>.<evento>`, em minúsculas. Para encaminhamento global, use `NATS_GLOBAL_ENABLED` e `NATS_GLOBAL_EVENTS`.

## WebSocket

A rota exige a **GLOBAL_API_KEY**, no parâmetro `token`, e aceita `instanceId` opcional:

```text
ws://localhost:8080/ws?token=GLOBAL_API_KEY&instanceId=UUID_DA_INSTANCIA
```

Sem `instanceId`, a conexão recebe os eventos encaminhados para clientes broadcast. Para receber eventos, habilite `websocketEnable` na instância.

O frame WebSocket é um envelope diferente do POST do webhook:

```json
{
  "queue": "uuid-da-instancia.message",
  "payload": "{\"event\":\"Message\",\"instanceId\":\"uuid-da-instancia\",\"data\":{}}"
}
```

`payload` é uma string JSON e precisa ser decodificada:

```javascript
const socket = new WebSocket(
  'ws://localhost:8080/ws?token=GLOBAL_API_KEY&instanceId=UUID_DA_INSTANCIA'
);
socket.onmessage = ({ data }) => {
  const frame = JSON.parse(data);
  const event = JSON.parse(frame.payload);
  console.log(frame.queue, event.event, event.data?.Info?.ID);
};
```

## Diagnóstico

Verifique, nesta ordem: token da instância, `subscribe`, transporte habilitado, destino acessível e logs de entrega. Os logs administrativos estão em `GET /instance/logs/:instanceId` e exigem `GLOBAL_API_KEY` no header `apikey`.

Um exemplo neste guia descreve o formato do emissor; não substitui um teste integrado com WhatsApp, destino HTTP ou broker real.

## Arquivamento de conversas

A categoria `CHAT_PRESENCE` inclui o evento `Archive`. Seu `data` contém `JID`, `Timestamp`, `Action` e `FromFullSync`. A sincronização inicial também pode produzir esses eventos; o servidor monta o payload a partir de `events.Archive`, sem converter o ponteiro do evento diretamente para um mapa.
