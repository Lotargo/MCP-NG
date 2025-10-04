# Файл: tests/test_human_input.py
# Назначение: Интерактивное тестирование инструмента 'human_input'.

import requests
import json

# --- Настройки ---
SERVER_URL = "http://localhost:8002"
TOOL_ENDPOINT = f"{SERVER_URL}/tools/run"

def run_test():
    """Основная функция для запуска интерактивного теста."""
    print("--- Тестирование инструмента: human_input (ИНТЕРАКТИВНЫЙ ТЕСТ) ---")

    test_prompt = "Вы уверены, что хотите продолжить опасную операцию? (да/нет)"
    expected_answer = "да"

    payload = {
        "name": "human_input",
        "arguments": {"prompt": test_prompt}
    }

    print("\n\033[94m[ИНСТРУКЦИЯ]\033[0m")
    print("1. Сейчас будет отправлен запрос на сервер, который 'зависнет' в ожидании.")
    print("2. Переключитесь в окно консоли, где запущен `mcp_server.py`.")
    print(f"3. Там появится вопрос: '{test_prompt}'")
    print(f"4. Введите в консоли сервера '{expected_answer}' и нажмите Enter.")
    print("5. Вернитесь в это окно, чтобы увидеть результат теста.")
    input("\nНажмите Enter, когда будете готовы начать...")

    try:
        print(f"\n[*] Отправляем запрос и ждем вашего ответа в консоли сервера...")
        # Используем большой таймаут, так как мы ждем ручного ввода
        response = requests.post(TOOL_ENDPOINT, json=payload, timeout=120)

        print(f"\n[*] Получен ответ от сервера! Статус-код: {response.status_code}")
        
        if response.status_code == 200:
            result_data = response.json()
            user_response = result_data.get("result", {}).get("user_response")

            if user_response == expected_answer:
                print(f"\033[92m[УСПЕХ]\033[0m Инструмент успешно получил ваш ответ: '{user_response}'")
            else:
                print(f"\033[91m[ОШИБКА]\033[0m Получен неожиданный ответ.")
                print(f"  Ожидалось: '{expected_answer}'")
                print(f"  Получено:  '{user_response}'")
        else:
            print(f"\033[91m[ОШИБКА]\033[0m Сервер вернул ошибку: {response.json()}")

    except requests.exceptions.ReadTimeout:
        print(f"\n\033[91m[ОШИБКА]\033[0m Время ожидания истекло. Вы не ответили на вопрос в консоли сервера.")
    except requests.exceptions.RequestException as e:
        print(f"\n\033[91m[КРИТИЧЕСКАЯ ОШИБКА]\033[0m Не удалось подключиться к серверу: {e}")

if __name__ == "__main__":
    run_test()