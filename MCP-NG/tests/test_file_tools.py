# Файл: tests/test_file_tools.py
# Назначение: Комплексное тестирование инструментов 'file_writer' и 'file_reader'.

import requests
import json
import os
from pathlib import Path

# --- Настройки ---
SERVER_URL = "http://localhost:8002"
TOOL_ENDPOINT = f"{SERVER_URL}/tools/run"

# Создаем временную папку для тестов, если ее нет
TEST_DIR = Path("temp_test_files")
TEST_DIR.mkdir(exist_ok=True)
TEST_FILE_PATH = TEST_DIR / "test_report.txt"
TEST_CONTENT = "Это тестовый отчет.\nСтрока 1.\nСтрока 2.\nПроверка кириллицы и спецсимволов: !@#$%^&*()"

def call_tool(payload: dict) -> dict:
    """Вспомогательная функция для вызова инструмента и обработки ответа."""
    try:
        response = requests.post(TOOL_ENDPOINT, json=payload, timeout=10)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        print(f"\n\033[91m[КРИТИЧЕСКАЯ ОШИБКА]\033[0m Не удалось подключиться к серверу: {e}")
        return {"critical_error": str(e)}

def run_test():
    """Основная функция для запуска последовательности тестов."""
    print("--- Комплексное тестирование: file_writer и file_reader ---")

    # === Тест 1: Запись файла ===
    print(f"\n[1] Тестируем 'file_writer'. Создаем файл: {TEST_FILE_PATH}")
    write_payload = {
        "name": "file_writer",
        "arguments": {"file_path": str(TEST_FILE_PATH), "content": TEST_CONTENT}
    }
    write_result = call_tool(write_payload)

    if "result" in write_result:
        print(f"\033[92m[УСПЕХ]\033[0m Инструмент 'file_writer' отработал: {write_result['result']}")
    else:
        print(f"\033[91m[ОШИБКА]\033[0m 'file_writer' вернул ошибку: {write_result.get('error', 'Неизвестная ошибка')}")
        return # Прерываем тест, если запись не удалась

    # === Тест 2: Чтение файла ===
    print(f"\n[2] Тестируем 'file_reader'. Читаем файл: {TEST_FILE_PATH}")
    read_payload = {
        "name": "file_reader",
        "arguments": {"file_path": str(TEST_FILE_PATH)}
    }
    read_result = call_tool(read_payload)

    if "result" not in read_result:
        print(f"\033[91m[ОШИБКА]\033[0m 'file_reader' вернул ошибку: {read_result.get('error', 'Неизвестная ошибка')}")
        return

    # === Тест 3: Проверка содержимого ===
    print("\n[3] Сравниваем записанное и прочитанное содержимое.")
    read_content = read_result["result"]
    if read_content == TEST_CONTENT:
        print("\033[92m[УСПЕХ]\033[0m Содержимое полностью совпадает!")
    else:
        print("\033[91m[ОШИБКА]\033[0m Содержимое НЕ совпадает!")
        print(f"Ожидалось: '{TEST_CONTENT}'")
        print(f"Получено:  '{read_content}'")

def cleanup():
    """Удаляет тестовые файлы и папки после завершения."""
    print("\n--- Очистка ---")
    if TEST_FILE_PATH.exists():
        try:
            TEST_FILE_PATH.unlink()
            print(f"[*] Тестовый файл '{TEST_FILE_PATH}' удален.")
            TEST_DIR.rmdir()
            print(f"[*] Тестовая папка '{TEST_DIR}' удалена.")
        except OSError as e:
            print(f"\033[91m[ОШИБКА]\033[0m Не удалось удалить тестовые файлы/папки: {e}")

if __name__ == "__main__":
    try:
        run_test()
    finally:
        cleanup()