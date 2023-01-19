#!/usr/bin/python3
import asyncio
import json
import websockets

CID = "fdaaaa9dcace802715dbb865eb784bf6b8aa48de9d8a425ca11a472edc72d240"

sub_events = {
  "container_id": "",
  "event": "subscribe",
  "type": "events"
}

sub_metrics = {
  "container_id": CID,
  "event": "subscribe",
  "type": "metrics"
}


async def run():
    async with websockets.connect("ws://localhost:8080/stream") as ws:
        print("[SND]", json.dumps(sub_events))
        await ws.send(json.dumps(sub_events))
        print("[SND]", json.dumps(sub_metrics))
        await ws.send(json.dumps(sub_metrics))

        while True:
            resp = await ws.recv()
            print("[RCV]", resp)


asyncio.get_event_loop().run_until_complete(run())
