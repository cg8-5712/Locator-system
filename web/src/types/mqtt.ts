export interface MQTTStatus {
  enabled: boolean;
  connected: boolean;
  topics: string[];
}

export interface MQTTMessage {
  topic: string;
  payload: string;
  qos: number;
  retained: boolean;
  received_at: string;
}

export interface MQTTMessageListResult {
  messages: MQTTMessage[];
}
