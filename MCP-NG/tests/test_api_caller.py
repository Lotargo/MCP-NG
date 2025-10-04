# Файл: tests/test_api_caller.py
# Назначение: Тестирование инструмента 'api_caller' с использованием JSONPlaceholder.

import requests
import json

# --- Настройки ---
SERVER_URL = "http://localhost:8002"
TOOL_ENDPOINT = f"{SERVER_URL}/tools/run"

def run_test():
    """
    Основная функция для запуска теста.
    """
    print("--- Тестирование инструмента: api_caller (с JSONPlaceholder) ---")

    # Используем супер-надежный и стандартный тестовый API JSONPlaceholder.
    # Этот URL возвращает один тестовый пост с id=1.
    test_api_url = "https://jsonplaceholder.typicode.com/posts/1"
    
    payload = {
        "name": "api_caller",
        "arguments": {
            "url": test_api_url,
            "method": "GET"
        }
    }

    print(f"[*] Отправляем запрос на {TOOL_ENDPOINT} со следующими данными:")
    print(json.dumps(payload, indent=2, ensure_ascii=False))

    try:
        # Отправляем POST-запрос на наш MCP-сервер
        response = requests.post(TOOL_ENDPOINT, json=payload, timeout=20)

        # Анализируем ответ
        print(f"\n[*] Получен ответ. Статус-код: {response.status_code}")

        if response.status_code == 200:
            result_data = response.json()
            
            # Проверяем, что ответ имеет ожидаемую структуру от JSONPlaceholder
            # (наличие ключей 'userId', 'id', 'title')
            if ("result" in result_data and 
                "userId" in result_data["result"] and 
                "id" in result_data["result"] and
                "title" in result_data["result"]):
                
                print("\033[92m[УСПЕХ]\033[0m Инструмент выполнен, и ответ от JSONPlaceholder имеет корректную структуру!")
                print("--- Результат выполнения ---")
                print(json.dumps(result_data, indent=2, ensure_ascii=False))
                print("--------------------------")
            else:
                 print(f"\033[91m[ОШИБКА]\033[0m Ответ от инструмента не содержит ожидаемых данных.")
                 print(json.dumps(result_data, indent=2, ensure_ascii=False))

        else:
            print(f"\033[91m[ОШИБКА]\033[0m Сервер вернул ошибку:")
            print(response.json())

    except requests.exceptions.RequestException as e:
        print(f"\033[91m[КРИТИЧЕСКАЯ ОШИБКА]\033[0m Не удалось подключиться к серверу: {e}")
        print("    Убедитесь, что mcp_server.py запущен.")

if __name__ == "__main__":
    run_test()