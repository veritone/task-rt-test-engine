# task-rt-test-engine
Lightweight engine to run with realtime heartbeat script. This engine demonstrates basic structure of an engine. It actually does not interact with any external APIs. The engine automatically generate fake outputs.

### Environment Variables

Variables that can be passed in as environment variables i.e. `docker run -e KAFKA_CHUNK_TOPIC="CHUNK_ALL"`

| Variable              | Description                                            |
|-----------------------|--------------------------------------------------------|
| KAFKA_BROKERS         | Comma-separated list of Kafka Broker addresses.        |
| KAFKA_CHUNK_TOPIC     | The Chunk Queue Kafka topic. Ex: "chunk_all"           |
| ENGINE_ID             | The engine ID                                          |
| ENGINE_INSTANCE_ID    | Unique instance ID for the engine instance             |
| KAFKA_INPUT_TOPIC     | The Kafka topic the engine should consume chunks from. |
| KAFKA_CONSUMER_GROUP  | The consumer group the engine must use.                |

### Sample kafka message with local test

```
{
    "type": "media_chunk",
    "timestampUTC": 1547173392785,
    "taskId": "19010211_MSLkayxxXrBWivV",
    "tdoId": "310781804",
    "jobId": "19010211_MSLkayxxXr",
    "chunkIndex": 5,
    "startOffsetMs": 4000,
    "endOffsetMs": 5000,
    "width": 1280,
    "height": 720,
    "mimeType": "image/jpeg",
    "cacheURI": "https://chunk-cache.s3.amazonaws.com/frames/310781804/1ad54583-eba7-4feb-af17-119f1b495345.jpg",
    "taskPayload": {
        "applicationId": "ed075985-bc94-406b-8639-44d1da42c3fb",
        "jobId": "19010211_MSLkayxxXr",
        "organizationId": "7682",
        "recordingId": "310781804",
        "taskId": "19010211_MSLkayxxXrBWivV",
        "taskPayload": {
            "organizationId": 7682
        },
        "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjb250ZW50QXBwbGljYXRpb25JZCI6ImVkMDc1OTg1LWJjOTQtNDA2Yi04NjM5LTQ0ZDFkYTQyYzNmYiIsImNvbnRlbnRPcmdhbml6YXRpb25JZCI6NzY4Miwic2NvcGUiOlt7ImFjdGlvbnMiOlsiYXNzZXQ6dXJpIiwiYXNzZXQ6YWxsIiwicmVjb3JkaW5nOnJlYWQiLCJyZWNvcmRpbmc6dXBkYXRlIl0sInJlc291cmNlcyI6eyJyZWNvcmRpbmdJZHMiOlsiMzEwNzgxODA0Il19fSx7ImFjdGlvbnMiOlsidGFzazp1cGRhdGUiXSwicmVzb3VyY2VzIjp7ImpvYklkcyI6WyIxOTAxMDIxMV9NU0xrYXl4eFhyIl0sInRhc2tJZHMiOlsiMTkwMTAyMTFfTVNMa2F5eHhYckJXaXZWIl0sInNvdXJjZUlkcyI6bnVsbH19XSwiaWF0IjoxNTQ3MTczMzQ1LCJleHAiOjE1NDc3NzgxNDUsInN1YiI6ImVuZ2luZS1ydW4iLCJqdGkiOiJmODZmMDYwOS1jNjFlLTQ0NWEtOWE2My1hOWM1NWNiOTIxNmUifQ.kB6NMG_xMS_7cmYbaRBYCaeFgOU1kwTJMfdyfR2DaXc",
        "veritoneApiBaseUrl": "https://api.veritone.com"
    },
    "chunkUUID": "554a4188-154d-4fbc-b121-3e2feb0edf20"
}
```

### Installation
```
make deps
```

### Run Local
```
go run main.go
```

### Building Engine
- Edit manifest.json, change engineId field to the engineID on VDA that you want to deploy
- Make sure to have the `.netrc` file -- see `.netrc.template` 

### Deploying Engine
Use the following links to go to VDA to deploy the engine

| env | link |
|-----|------|
|prod|https://developer.veritone.com/engines/c17ea304-377c-49b0-b667-2700042f94f3/builds|

### Running Engine
- Create task for engine to run:
  + Open graphQL console on the environment that you are running this engine
  + Using this template for graphQL command:
```graphql
  mutation createJob{
    createJob(input: {
      targetId: "371245784",
      tasks: [{
      engineId:"9e611ad7-2d3b-48f6-a51b-0a1ba40feab4", # The real-time adapter on the job
      payload:{
        url: "https://s3.amazonaws.com/test-chunk-engine/eng-usa.mp4"
        }
      },{
      engineId: "c17ea304-377c-49b0-b667-2700042f94f3",
      payload: {}
      }]
    }) {
    id
  }
}
```