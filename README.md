# go-micro-middleware

## Metrics middleware

```
// Setting the statsd endpoint with env vars in the main function
m := statsd.NewMetrics(
    metrics.Namespace("micro"),
    metrics.WithFields(metrics.Fields{
        "service": 'my.service.name',
    }),
    metrics.Collectors(
        os.Getenv("STATSD_HOST"),
    ),
)


// Setup the service with metrics wrappers for handlers and subscribers
service := micro.NewService(
    micro.Name('my.service.name'),
    micro.Server(
        server.NewServer(
            server.Name('my.service.name'),
            server.WrapHandler(middleware.MetricHandlerWrapper(m, time.Millisecond)),
            server.WrapSubscriber(middleware.MetricSubscriberWrapper(m, time.Millisecond)),
        ),
    ),
)
```

## Logging middleware

```
service := micro.NewService(
    micro.Name('my.service.name'),
    micro.Server(
        server.NewServer(
            server.Name('my.service.name'),
            server.WrapHandler(middleware.LogHandlerWrapper),
            server.WrapSubscriber(middleware.LogSubscriberWrapper),
        ),
    ),
)
```


## Trace middleware

```
service := micro.NewService(
	micro.Name('my.service.name'),
	micro.Client(client.NewClient(
		client.Wrap(middleware.TraceWrap),
		client.Wrap(middleware.LogWrap),
	)),
)
```

## Datadog + Kubernetes

If you are running the Datadog agents on Kubernetes in the same namespace as your micro services.  You can create a kubernetes service template for the dd-agent and push metrics via the statsd port with all your kubernetes metrics.

Get your dd-agent.yaml daemon set configuration from here, https://app.datadoghq.com/account/settings#agent/kubernetes

You will need to configure a kube service so your go-micro services can connect to the dd-agent pods.  A tradeoff of using a service is you may not connect to the same node. Host ports maybe a better option.

```
---
apiVersion: v1
kind: Service
metadata:
  name: dd-agent-service
  labels:
    app: dd-agent
spec:
  ports:
    - port: 8125
      targetPort: 8125
      protocol: UDP
  selector:
    app: dd-agent
```
