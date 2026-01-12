import type { Envelope, WebSocketClient } from "./websocket-client";
import { z } from "zod";

type ZodInfer<T> = T extends z.ZodType<infer U> ? U : never;

type EnvelopeObj<T extends { [key: string]: z.ZodType<any> }> = {
  [K in keyof T]: K extends string ? Envelope<K, ZodInfer<T[K]>> : never;
}[keyof T];

export class MessageReceipt {
  constructor(
    private client: WebSocketClient,
    readonly id: string,
  ) {}

  response(): Promise<Envelope> {
    return new Promise<Envelope>((resolve) => {
      const cancel = this.client.addMessageHandler((envelope) => {
        if (envelope.reply_to === this.id) {
          cancel();
          resolve(envelope as Envelope);
        }
      });
    });
  }

  async responseSchema<T = any>(
    schema: z.ZodType<T>,
  ): Promise<Envelope<string, T>> {
    const res = await this.response();
    return {
      ...res,
      payload: schema.parse(res.payload),
    };
  }

  async responseTypes<T extends { [key: string]: z.ZodType<any> }>(
    types: T,
  ): Promise<EnvelopeObj<T>> {
    const res = await this.response();
    const t = types[res.type];
    if (!t) {
      throw new Error(`Unexpected response type: ${res.type}`);
    }

    return {
      ...res,
      type: res.type,
      payload: t.parse(res.payload),
    } as EnvelopeObj<T>;
  }
}
