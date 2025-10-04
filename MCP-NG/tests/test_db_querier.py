# Файл: tests/test_db_querier.py
# Назначение: Тестирование инструмента 'db_querier'.

import requests
import json
import sqlite3
from pathlib import Path

# --- Настройки ---
SERVER_URL = "http://localhost:8002"
TOOL_ENDPOINT = f"{SERVER_URL}/tools/run"
TEST_DB_PATH = Path("test_database.sqlite")

def setup_test_database():
    """Создает временную БД SQLite с тестовыми данными."""
    print("--- Подготовка тестовой базы данных ---")
    if TEST_DB_PATH.exists():
        TEST_DB_PATH.unlink() # Очищаем от предыдущих запусков
    
    con = sqlite3.connect(TEST_DB_PATH)
    cur = con.cursor()
    # Создаем таблицу
    cur.execute("CREATE TABLE users(id INTEGER PRIMARY KEY, name TEXT, role TEXT, age INTEGER)")
    # Вставляем данные
    cur.execute("INSERT INTO users VALUES (1, 'Alice', 'admin', 30)")
    cur.execute("INSERT INTO users VALUES (2, 'Bob', 'user', 25)")
    cur.execute("INSERT INTO users VALUES (3, 'Charlie', 'user', 35)")
    con.commit()
    con.close()
    print(f"[*] Временная БД '{TEST_DB_PATH}' создана и наполнена данными.")

def cleanup_test_database():
    """Удаляет временную БД."""
    print("\n--- Очистка тестовой базы данных ---")
    if TEST_DB_PATH.exists():
        TEST_DB_PATH.unlink()
        print(f"[*] Временная БД '{TEST_DB_PATH}' удалена.")

def run_test():
    """Основная функция для запуска теста."""
    print("\n--- Тестирование инструмента: db_querier ---")
    
    # SQL-запрос, который мы хотим выполнить
    sql_query = "SELECT name, role FROM users WHERE age > 28 ORDER BY name"

    payload = {
        "name": "db_querier",
        "arguments": {
            "db_path": str(TEST_DB_PATH),
            "query": sql_query
        }
    }
    print(f"[*] Выполняем SQL-запрос: '{sql_query}'")

    try:
        response = requests.post(TOOL_ENDPOINT, json=payload, timeout=10)
        response.raise_for_status()
        result_data = response.json()

        print(f"[*] Получен ответ. Статус-код: {response.status_code}")

        if "result" in result_data:
            # Ожидаемый результат: Alice (30) и Charlie (35)
            expected_result = [
                {"name": "Alice", "role": "admin"},
                {"name": "Charlie", "role": "user"}
            ]
            if result_data["result"] == expected_result:
                print("\033[92m[УСПЕХ]\033[0m Инструмент вернул корректные данные из БД!")
                print("--- Полученные данные ---")
                print(json.dumps(result_data, indent=2, ensure_ascii=False))
                print("-------------------------")
            else:
                print("\033[91m[ОШИБКА]\03T[0m Данные из БД не соответствуют ожидаемым.")
                print("Ожидалось:", json.dumps({"result": expected_result}, indent=2))
                print("Получено:", json.dumps(result_data, indent=2))
        else:
            print(f"\033[91m[ОШИБКА]\033[0m Инструмент вернул ошибку: {result_data.get('error')}")

    except requests.exceptions.RequestException as e:
        print(f"\033[91m[КРИТИЧЕСКАЯ ОШИБКА]\033[0m Не удалось подключиться к серверу: {e}")

if __name__ == "__main__":
    try:
        setup_test_database()
        run_test()
    finally:
        cleanup_test_database()