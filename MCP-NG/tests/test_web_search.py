# Файл: tests/test_web_search.py
# Назначение: Тестирование инструмента 'web_search' через API нашего MCP-сервера.

import requests
import json

# --- Настройки ---
SERVER_URL = "http://localhost:8002"
TOOL_ENDPOINT = f"{SERVER_URL}/tools/run"

def run_test():
    """
    Основная функция для запуска теста.
    """
    print("--- Тестирование инструмента: web_search ---")

    # 1. Формируем полезную нагрузку (payload) для нашего запроса.
    #    Она должна точно соответствовать формату, который ожидает сервер.
    payload = {
        "name": "web_search",
        "arguments": {
            "query": "Что такое квантовая запутанность простыми словами?",
            "max_results": 2
        }
    }

    print(f"[*] Отправляем запрос на {TOOL_ENDPOINT} со следующими данными:")
    print(json.dumps(payload, indent=2, ensure_ascii=False))

    try:
        # 2. Отправляем POST-запрос на сервер
        response = requests.post(TOOL_ENDPOINT, json=payload, timeout=20) # таймаут 20с

        # 3. Анализируем ответ от сервера
        print(f"\n[*] Получен ответ. Статус-код: {response.status_code}")

        if response.status_code == 200:
            # Если все успешно, выводим результат
            result_data = response.json()
            print("\033[92m[УСПЕХ]\033[0m Инструмент выполнен успешно!")
            print("--- Результат выполнения ---")
            # Используем json.dumps для красивого вывода
            print(json.dumps(result_data, indent=2, ensure_ascii=False))
            print("--------------------------")
        else:
            # Если сервер вернул ошибку, выводим ее
            print(f"\033[91m[ОШИБКА]\033[0m Сервер вернул ошибку:")
            try:
                print(response.json())
            except json.JSONDecodeError:
                print(response.text)

    except requests.exceptions.RequestException as e:
        print(f"\033[91m[КРИТИЧЕСКАЯ ОШИБКА]\033[0m Не удалось подключиться к серверу: {e}")
        print("    Убедитесь, что mcp_server.py запущен.")

if __name__ == "__main__":
    run_test()