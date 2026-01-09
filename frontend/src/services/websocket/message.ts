import type { WebSocketClient, EnvelopeSchema } from "./websocket-client";

export class MessageReceipt {
  constructor(
    private client: WebSocketClient,
    readonly id: string,
  ) {}

  response(schema?: any): Promise<EnvelopeSchema> {
    return new Promise<EnvelopeSchema>((resolve) => {
      const cancel = this.client.addMessageHandler((envelope) => {
        if (envelope.reply_to === this.id) {
          cancel();
          resolve(envelope);
        }
      });
    });
  }
}
