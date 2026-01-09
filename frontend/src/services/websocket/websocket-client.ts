import { jsonCodec } from "@/lib/utils";
import z from "zod";
import { v4 as uuid } from "uuid";
import { MessageReceipt } from "./message";

export type ConnectionState =
  | "disconnected"
  | "connecting"
  | "connected"
  | "reconnecting";

const envelopeSchema = jsonCodec(
  z.object({
    type: z.string(),
    id: z.uuid(),
    reply_to: z.uuid().optional(),
    idempotency_key: z.string().optional(),
    subscription_id: z.uuid().optional(),
    seq: z.number().optional(),
    time: z.iso.datetime(),
    payload: z.unknown().optional(),
  }),
);

export type EnvelopeSchema = z.infer<typeof envelopeSchema>;

export interface EnvelopeError {
  code: string;
  message: string;
  detail?: unknown;
}

export type MessageHandler = (envelope: EnvelopeSchema) => void;
export type ConnectionStateHandler = (state: ConnectionState) => void;
export type AuthFailureHandler = () => void;

interface WebSocketClientOptions {
  /**
   * Function to get a fresh auth token. Called on reconnect.
   */
  getToken: () => Promise<string | null>;
  /**
   * Initial retry delay in milliseconds. Default: 1000ms.
   */
  initialRetryDelay?: number;
  /**
   * Maximum retry delay in milliseconds. Default: 30000ms (30s).
   */
  maxRetryDelay?: number;
  /**
   * Maximum number of retry attempts before giving up. Default: Infinity.
   */
  maxRetries?: number;
}

const DEFAULT_INITIAL_RETRY_DELAY = 1000;
const DEFAULT_MAX_RETRY_DELAY = 30000;
const DEFAULT_MAX_RETRIES = Infinity;

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private state: ConnectionState = "disconnected";
  private retryCount = 0;
  private retryTimeout: ReturnType<typeof setTimeout> | null = null;
  private messageHandlers = new Set<MessageHandler>();
  private stateHandlers = new Set<ConnectionStateHandler>();
  private authFailureHandlers = new Set<AuthFailureHandler>();
  private seq = 1;

  private readonly options: Required<
    Pick<
      WebSocketClientOptions,
      "getToken" | "initialRetryDelay" | "maxRetryDelay" | "maxRetries"
    >
  >;

  constructor(options: WebSocketClientOptions) {
    this.options = {
      initialRetryDelay:
        options.initialRetryDelay ?? DEFAULT_INITIAL_RETRY_DELAY,
      maxRetryDelay: options.maxRetryDelay ?? DEFAULT_MAX_RETRY_DELAY,
      maxRetries: options.maxRetries ?? DEFAULT_MAX_RETRIES,
      getToken: options.getToken,
    };
  }

  /**
   * Get the current connection state.
   */
  getState(): ConnectionState {
    return this.state;
  }

  /**
   * Subscribe to connection state changes.
   * Returns an unsubscribe function.
   */
  subscribeToState(handler: ConnectionStateHandler): () => void {
    this.stateHandlers.add(handler);
    return () => this.stateHandlers.delete(handler);
  }

  /**
   * Subscribe to auth failure events.
   * Returns an unsubscribe function.
   */
  subscribeToAuthFailure(handler: AuthFailureHandler): () => void {
    this.authFailureHandlers.add(handler);
    return () => this.authFailureHandlers.delete(handler);
  }

  /**
   * Add a message handler.
   * Returns an unsubscribe function.
   */
  addMessageHandler(handler: MessageHandler): () => void {
    this.messageHandlers.add(handler);
    return () => this.messageHandlers.delete(handler);
  }

  /**
   * Connect to the WebSocket server.
   */
  async connect(): Promise<void> {
    if (this.state === "connected" || this.state === "connecting") {
      return;
    }

    this.setState(this.retryCount > 0 ? "reconnecting" : "connecting");

    const token = await this.options.getToken();
    if (!token) {
      this.setState("disconnected");
      this.notifyAuthFailure();
      return;
    }

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${window.location.host}/api/ws?token=${encodeURIComponent(token)}`;

    try {
      this.ws = new WebSocket(wsUrl);
      this.setupEventHandlers();
    } catch (error) {
      console.error("WebSocket connection error:", error);
      this.scheduleReconnect();
    }
  }

  /**
   * Disconnect from the WebSocket server.
   */
  disconnect(): void {
    this.clearRetryTimeout();
    this.retryCount = 0;

    if (this.ws) {
      this.ws.onclose = null; // Prevent reconnect on manual disconnect
      this.ws.close(1000, "Client disconnect");
      this.ws = null;
    }

    this.setState("disconnected");
  }

  send(
    type: string,
    payload: any,
    idempotencyKey?: string,
    replyTo?: string,
  ): MessageReceipt {
    if (this.getState() !== "connected") {
      console.warn("Cannot send message, WebSocket not connected");
      throw new Error("WebSocket not connected");
    }

    const envelope: EnvelopeSchema = {
      type,
      id: uuid(),
      seq: this.seq++,
      time: new Date().toISOString(),
      payload,
    };

    if (idempotencyKey) {
      envelope.idempotency_key = idempotencyKey;
    }
    if (replyTo) {
      envelope.reply_to = replyTo;
    }

    const json = envelopeSchema.encode(envelope);
    this.ws!.send(json);

    return new MessageReceipt(this, envelope.id);
  }

  private setupEventHandlers(): void {
    if (!this.ws) return;

    this.ws.onopen = () => {
      console.log("WebSocket connected");
      this.retryCount = 0;
      this.setState("connected");
    };

    this.ws.onclose = (event) => {
      console.log("WebSocket closed:", event.code, event.reason);
      this.scheduleReconnect();
    };

    this.ws.onerror = (error) => {
      console.error("WebSocket error:", error);
    };

    this.ws.onmessage = (event) => {
      try {
        const envelope = envelopeSchema.decode(event.data);
        this.handleMessage(envelope);
      } catch (error) {
        console.error("Failed to parse WebSocket message:", error);
      }
    };
  }

  private handleMessage(envelope: EnvelopeSchema): void {
    // Handle "Bye" message indicating auth expiration
    if (envelope.type === "connection.close") {
      console.log("Received close message, reconnecting with fresh token...");
      this.ws?.close(1000, "Server requested disconnect");
      return;
    }

    // Broadcast to all message handlers
    this.messageHandlers.forEach((handler) => {
      try {
        handler(envelope);
      } catch (error) {
        console.error("Message handler error:", error);
      }
    });
  }

  private scheduleReconnect(): void {
    if (this.retryCount >= this.options.maxRetries) {
      console.log("Max retries reached, giving up");
      this.setState("disconnected");
      return;
    }

    this.setState("reconnecting");
    this.clearRetryTimeout();

    const delay = Math.min(
      this.options.initialRetryDelay * Math.pow(2, this.retryCount),
      this.options.maxRetryDelay,
    );

    console.log(
      `Reconnecting in ${delay}ms (attempt ${this.retryCount + 1})...`,
    );

    this.retryTimeout = setTimeout(() => {
      this.retryCount++;
      this.connect();
    }, delay);
  }

  private clearRetryTimeout(): void {
    if (this.retryTimeout) {
      clearTimeout(this.retryTimeout);
      this.retryTimeout = null;
    }
  }

  private setState(state: ConnectionState): void {
    if (this.state !== state) {
      this.state = state;
      this.stateHandlers.forEach((handler) => handler(state));
    }
  }

  private notifyAuthFailure(): void {
    this.authFailureHandlers.forEach((handler) => {
      try {
        handler();
      } catch (error) {
        console.error("Auth failure handler error:", error);
      }
    });
  }
}
