import argparse
from os import getenv

import pika
import wagglemsg
from waggle.plugin.time import get_timestamp

def main(args):
    if args.type in ["int", "i"]:
        v = int(args.value)
    elif args.type in ["float", "f"]:
        v = float(args.value)
    elif args.type in ["string", "str"]:
        v = str(args.value)
    else:
        raise Exception(f'Wrong type detected: {args.type}')
    
    meta = {}
    for m in args.meta:
        try:
            sp = m.split('=')
            meta[sp[0]] = sp[1]
        except Exception as ex:
            raise Exception(f'Failed to parser meta {args.meta}: {str(ex)}')
    meta.update({
        "node": "plugin",
        "vsn": "W000",
    })
    msg = wagglemsg.Message(
        name=args.topic,
        value=v,
        timestamp=get_timestamp(),
        meta=meta,
    )

    params = pika.ConnectionParameters(
        host=args.rabbitmq_host,
        port=args.rabbitmq_port,
        credentials=pika.PlainCredentials("plugin", "plugin"),
        retry_delay=60,
        socket_timeout=10.0
    )

    conn = pika.BlockingConnection(params)
    ch = conn.channel()

    ch.basic_publish(
        "waggle.msg",
        args.topic,
        wagglemsg.dump(msg),
        properties=pika.BasicProperties(
            delivery_mode=2,
            user_id="plugin",
        )
    )
    print("message published")
    ch.close()
    conn.close()

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--rabbitmq_host",default=getenv("RABBITMQ_HOST", "localhost"))
    parser.add_argument("--rabbitmq_port", default=getenv("RABBITMQ_PORT", "5672"), type=int)
    parser.add_argument("--rabbitmq_username", default=getenv("RABBITMQ_USERNAME", ""))
    parser.add_argument("topic", help="Name of the topic")
    parser.add_argument("type", help="Type of value either in string, int, and float")
    parser.add_argument("value", help="Value")
    parser.add_argument("--meta", "-m", action="append", default=[], help="Meta information to be added")
    main(parser.parse_args())