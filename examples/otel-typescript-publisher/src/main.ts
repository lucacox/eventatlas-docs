import {
  SpanKind,
  SpanStatusCode,
  trace,
  type Span,
} from "@opentelemetry/api";
import { OTLPTraceExporter } from "@opentelemetry/exporter-trace-otlp-proto";
import { resourceFromAttributes } from "@opentelemetry/resources";
import {
  AlwaysOnSampler,
  BatchSpanProcessor,
} from "@opentelemetry/sdk-trace-base";
import { NodeTracerProvider } from "@opentelemetry/sdk-trace-node";
import {
  connect,
  type NatsConnection,
} from "@nats-io/transport-node";

const defaults = {
  natsUrl: "nats://127.0.0.1:4222",
  serviceName: "orders-typescript-publisher",
  serviceNamespace: "eventatlas.examples",
  environment: "development",
  subject: "orders.created",
  subjectTemplate: "orders.*",
  message: '{"event":"order.created","order_id":"demo-order-123"}',
} as const;

interface Config {
  natsUrl: string;
  serviceName: string;
  serviceNamespace: string;
  environment: string;
  subject: string;
  subjectTemplate: string;
  message: string;
}

const config = loadConfig();
const provider = new NodeTracerProvider({
  resource: resourceFromAttributes({
    "service.name": config.serviceName,
    "service.namespace": config.serviceNamespace,
    "deployment.environment.name": config.environment,
  }),
  sampler: new AlwaysOnSampler(),
  spanProcessors: [
    new BatchSpanProcessor(new OTLPTraceExporter()),
  ],
});

provider.register();

async function main(): Promise<void> {
  let connection: NatsConnection | undefined;
  try {
    connection = await connect({
      servers: config.natsUrl,
      name: config.serviceName,
      timeout: 5_000,
    });

    await publish(connection, config);
    console.log(
      `published ${JSON.stringify(config.message)} to ${config.subject} as service ${config.serviceName}; ` +
        "the span will be exported through OTLP/HTTP",
    );
  } finally {
    try {
      await connection?.drain();
    } finally {
      await provider.shutdown();
    }
  }
}

async function publish(connection: NatsConnection, config: Config): Promise<void> {
  const tracer = trace.getTracer(
    "github.com/lucacox/eventatlas-docs/examples/otel-typescript-publisher",
  );

  await tracer.startActiveSpan(
    `${config.subject} send`,
    {
      kind: SpanKind.PRODUCER,
      attributes: {
        "messaging.system": "nats",
        "messaging.operation.type": "send",
        "messaging.destination.name": config.subject,
        "messaging.destination.template": config.subjectTemplate,
      },
    },
    async (span) => publishWithinSpan(connection, config, span),
  );
}

async function publishWithinSpan(
  connection: NatsConnection,
  config: Config,
  span: Span,
): Promise<void> {
  try {
    connection.publish(config.subject, config.message);
    await connection.flush();
    span.setStatus({ code: SpanStatusCode.OK, message: "message published" });
  } catch (error: unknown) {
    const publishError = error instanceof Error ? error : new Error(String(error));
    span.recordException(publishError);
    span.setStatus({ code: SpanStatusCode.ERROR, message: publishError.message });
    throw publishError;
  } finally {
    span.end();
  }
}

function loadConfig(): Config {
  return {
    natsUrl: envOrDefault("NATS_URL", defaults.natsUrl),
    serviceName: envOrDefault("OTEL_SERVICE_NAME", defaults.serviceName),
    serviceNamespace: envOrDefault(
      "EXAMPLE_SERVICE_NAMESPACE",
      defaults.serviceNamespace,
    ),
    environment: envOrDefault("EXAMPLE_ENVIRONMENT", defaults.environment),
    subject: envOrDefault("NATS_SUBJECT", defaults.subject),
    subjectTemplate: envOrDefault(
      "NATS_SUBJECT_TEMPLATE",
      defaults.subjectTemplate,
    ),
    message: envOrDefault("MESSAGE_BODY", defaults.message),
  };
}

function envOrDefault(name: string, fallback: string): string {
  const value = process.env[name]?.trim();
  return value && value.length > 0 ? value : fallback;
}

void main().catch((error: unknown) => {
  console.error(error);
  process.exitCode = 1;
});
