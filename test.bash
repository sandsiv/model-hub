#!/bin/bash

# Создаем временный файл с JSON телом запроса
json_data=$(mktemp)
echo '{"parameters":{"model": "model2"}}' > "$json_data"

# Функция для отправки POST запроса и вывода сообщения
do_post() {
  response=$(curl -s -w "%{http_code}" -X POST -H "Content-Type: application/json" -d @"$json_data" http://127.0.0.1:7766/predict)
  http_status=$(tail -n1 <<< "$response")  # Получаем HTTP статус из ответа
  echo "Ответ сервера с HTTP статусом: $http_status"
}

export -f do_post
export json_data

# Используем seq для генерации чисел от 1 до 100 и передаем их в xargs
seq 100 | xargs -n1 -P100 -I {} bash -c 'do_post'

# Удаляем временный файл
rm "$json_data"
