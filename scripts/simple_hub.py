#!/usr/bin/python3
import json
import asyncio
import websockets
import time

CID = "e63d12ed9e74cb2d5994e9e0356aad7588939108fb2a1e5ec729274e07e820bf"

subscribe = {
  "container_id": CID,
  "event": "subscribe",
  "type": "metrics"
}

unsubscribe = {
  "container_id": CID,
  "event": "unsubscribe",
  "type": "metrics"
}


async def run():
    sub = json.dumps(subscribe)
    usub = json.dumps(unsubscribe)

    async with websockets.connect("ws://localhost:8081/stream") as ws:
        # subscribe
        await ws.send(sub)
        print("[OUT]", sub)

        start = time.time()
        while True:
            now = time.time()
            if (now - start) > 5:
                await ws.send(usub)
                print("[OUT]", usub)
                break

            rcv = await ws.recv()
            print("[IN]", rcv)


if __name__ == "__main__":
    asyncio.get_event_loop().run_until_complete(run())