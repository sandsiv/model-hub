import argparse
import json
import sys
import os
import importlib.util
import threading
import logging
import time

import requests
from http.server import BaseHTTPRequestHandler, HTTPServer

parser = argparse.ArgumentParser()
parser.add_argument('worker_id', type=str)
parser.add_argument('path', type=str)
parser.add_argument('port', type=int)
parser.add_argument('handler_path', type=str)
args = parser.parse_args()

logging.basicConfig(format=f'[%(levelname)s][Python worker][{args.worker_id}] %(message)s', level=logging.INFO)


class RequestHandler(BaseHTTPRequestHandler):
    def send_error(self, code, message=None):
        self.send_response(code)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        error_message = {
            'error': message if message else f"Error code: {code}"
        }
        self.wfile.write(json.dumps(error_message).encode('utf-8'))

    def do_POST(self):
        content_length = int(self.headers['Content-Length'])
        post_data = self.rfile.read(content_length)

        if self.path == '/predict':
            if not handler.model_loaded:
                logging.error('Model not loaded')
                self.send_error(500, 'Model not loaded')
                return

            try:
                data = json.loads(post_data.decode('utf-8'))
            except json.JSONDecodeError:
                logging.error('Invalid request JSON data')
                self.send_error(400, 'Invalid request JSON data')
                return
            try:
                prediction = handler.predict(data)
            except Exception as e:
                self.send_response(500)
                self.send_header('Content-type', 'application/json')
                self.end_headers()
                self.wfile.write(str(e).encode('utf-8'))
                return
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps(prediction).encode('utf-8'))

        else:
            logging.error('Invalid endpoint')
            self.send_error(404, 'Invalid endpoint')


if __name__ == '__main__':
    logging.info(f'Starting worker {args.worker_id}')
    handler_path = os.path.abspath(args.handler_path)
    if not os.path.isfile(handler_path):
        logging.error(f"{args.handler_path} does not exist or is not a file")
        sys.exit(1)

    spec = importlib.util.spec_from_file_location('handler', handler_path)
    handler_module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(handler_module)

    handler = handler_module.Handler()

    logging.info("Start loading model")
    try:
        handler.load_model(args.path)
    except Exception as load_model_exception:
        logging.error(f"Failed to load model: {load_model_exception}")
        sys.exit(1)
    logging.info("Model loaded")


    def notify_ready():
        time.sleep(1)
        port = os.getenv("SERVER_PORT", default="7766")
        model_ready_url = f"http://127.0.0.1:{port}/model-ready"
        model_ready_payload = {"worker_id": args.worker_id}
        try:
            requests.post(model_ready_url, json=model_ready_payload, timeout=500)
        except Exception as notify_ready_exception:
            logging.error(f"Failed to send model ready notification: {notify_ready_exception}")


    load_thread = threading.Thread(target=notify_ready, name='ReadyNotifier')
    load_thread.start()
    try:
        server = HTTPServer(('127.0.0.1', args.port), RequestHandler)
        logging.info(f'Python worker REST started at http://127.0.0.1:{args.port}')
        server.serve_forever()
    except KeyboardInterrupt:
        logging.info('Stopping server...')
        server.socket.close()
        sys.exit(0)
