![](/Users/przemek/Downloads/connection-icon-19.jpg)

# Mesh

[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0)

* EventMesh
* Trans-protocol converter
* Inter-protol bridge/proxy
* Meta Broker

## Features:

* async only
* lightweight, fast
* throttling
* metrics: prometheus
* js support
* web console
* Linux/Mac/Windows (single executable)
* Docker

## Protocols:

- HTTP
- Kafka
- AMQP 1.0
- MQTT
- Websockets

## Messsage model 

* Data - array of bytes
* Headers - map string -> array of bytes 

| Protocol      | Message       | Headers  |
| ------------- |:-------------:| -----:|
|1|1|1

## Components

### HTTP

#### Consumer 

HTTP body -> Message.Data

HTTP headers -> Message.Headers

#### Producer

Message.Data -> HTTP body

Message.Headers -> HTTP headers

### Kafka

#### Consumer

Message key -> Message.Headers["kafka_key"]

Message data -> Message.Data 

#### Producer

Message.Headers["kafka_key"] -> Kafka Message Key

Message.Data -> Kafka Message 

### AMQP 1.0

### MQTT

### Websockets

## Security

## Contributing

There is a lot to do. If you want to help me, you are welcome.