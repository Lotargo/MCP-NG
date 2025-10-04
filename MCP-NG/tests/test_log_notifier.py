# Файл: tests/test_log_notifier.py
# Назначение: Тестирование инструмента 'log_notifier'.

import requests
import json
import random
from pathlib import Path

# --- Настройки ---
SERVER_URL = "http://localhost:8002"
TOOL_ENDPOINT = f"{SERVER_URL}/tools/run"
LOG_FILE = Path("notifications.log")

def run_test():
    """Основная функция для запуска теста."""
    print("--- Тестирование инструмента: log_notifier ---")

    # Генерируем уникальный маркер для этого тестового запуска
    test_marker = f"TEST_RUN_{random.randint(10000, 99999)}"
    test_message = f"Задача успешно завершена. Маркер: {test_marker}"

    payload = {
        "name": "log_notifier",
        "arguments": {
            "message": test_message,
            "level": "SUCCESS"
        }
    }

    print(f"[*] Отправляем уведомление с уникальным маркером: '{test_marker}'")

    try:
        # 1. Вызываем инструмент через сервер
        response = requests.post(TOOL_ENDPOINT, json=payload, timeout=10)
        response.raise_for_status()
        result_data = response.json()

        if "error" in result_data:
            print(f"\033[91m[ОШИБКА]\033[0m Инструмент вернул ошибку: {result_data['error']}")
            return

        print(f"[*] Инструмент отработал успешно: {result_data.get('result')}")

        # 2. Проверяем результат - читаем лог-файл
        print(f"[*] Проверяем содержимое файла '{LOG_FILE}'...")
        if not LOG_FILE.is_file():
            print(f"\033[91m[ОШИБКА]\033[0m Лог-файл '{LOG_FILE}' не был создан.")
            return

        with open(LOG_FILE, 'r', encoding='utf-8') as f:
            log_content = f.read()

        # 3. Ищем наш уникальный маркер в содержимом файла
        if test_marker in log_content:
            print(f"\033[92m[УСПЕХ]\033[0m Уникальный маркер '{test_marker}' найден в лог-файле!")
        else:
            print(f"\033[91m[ОШИБКА]\033[0m Уникальный маркер '{test_marker}' НЕ найден в лог-файле.")

    except requests.exceptions.RequestException as e:
        print(f"\033[91m[КРИТИЧЕСКАЯ ОШИБКА]\033[0m Не удалось подключиться к серверу: {e}")
    
    finally:
        # Опционально: можно удалить лог-файл после теста, но для отладки полезно его оставить.
        # if LOG_FILE.exists():
        #     LOG_FILE.unlink()
        pass

if __name__ == "__main__":
    run_test()