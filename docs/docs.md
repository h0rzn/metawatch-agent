# Documentation

## API
#### /login
Request [POST]
```
{
    "password": "master",
    "username": "master"
}
```

Response
```
{
    "code": 200,
    "expire": "2023-01-19T10:54:40+01:00",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2NzQxMjIwODAsImlkZW50aXR5IjoibWFzdGVyIiwib3JpZ19pYXQiOjE2NzQxMTg0ODB9.fOCdkCef01Svl1NFhKli9FlxQvtAcCYeE9NV67e8L5k"
}
```
#### [JWT] /api/refresh_token

#### [JWT] /api/containers/:id
#### [JWT] /api/containers/:id/metrics?from=X&to=Y
#### [JWT] /api/containers/all

#### [JWT] /api/images/all
#### [JWT] /api/image/:id

#### [JWT] /api/about
#### [JWT] /api/volumes
```
{
  "version": "20.10.21",
  "api_version": "1.41",
  "os": "linux",
  "image_n": 12,
  "container_n": 7
}
```

## Hub
- resources will be deleted after 10min if not used (no subs)

`/stream`
Subscribe to Resource
> `container_id`: id of container
`event`: `subscribe` || `unsubscribe`
`type`: `metrics` || `logs` || `events`

```
{
  "container_id": <cid>, 
  "event": "subscribe,
  "type": "metrics"
}
```

### Generic Resource (metrics, logs)
Subscribe
```
{
  "container_id": <cid>, 
  "event": "subscribe,
  "type": "metrics" // "type": "logs"
}
```
Response
```
{
   "container_id":"fdaaaa9dcace802715dbb865eb784bf6b8aa48de9d8a425ca11a472edc72d240",
   "type":"metrics_set",
   "message":{
      "when":"2023-01-09T21:02:17.414+01:00",
      "cpu":{
         "perc":0.04666666666666667,
         "online":4
      },
      "memory":{
         "perc":0.012338222519843144,
         "usage_bytes":1015808,
         "available_bytes":8233017344
      },
      "disk":{
         "read":0,
         "write":0
      },
      "net":{
         "in":1226,
         "out":0
      }
   }
}
```
### Events Resource (events)
Subscribe
```
{
  "container_id": <cid>, 
  "event": "subscribe,
  "type": "events"
}
```
```
{
    "type": "event",
    "message": {
        "type": "container_start",
        "id": <id of referenced item>
    }
}
```
on `container_start` `id` would be container id, for image events the image id, ... 

### Combined Metrics (metrics of all running container summed up)
Subscribe
```
{
  "container_id": "_all", 
  "event": "subscribe,
  "type": "combined_metrics"
}
```
Response
```
{
   "container_id":"_all",
   "type":"combined_metrics",
   "message":{
      "when":"2023-01-09T21:02:17.414+01:00",
      "cpu":{
         "perc":0.04666666666666667,
         "online":4
      },
      "memory":{
         "perc":0.012338222519843144,
         "usage_bytes":1015808,
         "available_bytes":8233017344
      },
      "disk":{
         "read":0,
         "write":0
      },
      "net":{
         "in":1226,
         "out":0
      }
   }
}
```