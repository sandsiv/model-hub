import time


class Handler:
    def __init__(self):
        self.model_loaded = False
        self.model = None

    def load_model(self, path):
        time.sleep(5)
        self.model_loaded = True

    def predict(self, data):
        prediction = {'predictions': data}
        return prediction
