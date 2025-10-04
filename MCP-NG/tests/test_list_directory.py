# Файл: tests/test_list_directory.py
# Назначение: Тестирование инструмента 'list_directory'.

import requests
import json
import shutil
from pathlib import Path

# --- Настройки ---
SERVER_URL = "http://localhost:8002"
TOOL_ENDPOINT = f"{SERVER_URL}/tools/run"

# Создаем сложную временную структуру для теста
TEST_ROOT = Path("temp_list_test")
TEST_SUBDIR = TEST_ROOT / "subdir"
TEST_FILES = [TEST_ROOT / "report.txt", TEST_ROOT / "data.csv", TEST_SUBDIR / "notes.txt"]

def setup_test_environment():
    """Создает тестовые файлы и папки."""
    print("--- Подготовка тестового окружения ---")
    if TEST_ROOT.exists():
        shutil.rmtree(TEST_ROOT) # Очищаем от предыдущих запусков
    TEST_SUBDIR.mkdir(parents=True, exist_ok=True)
    for f in TEST_FILES:
        f.touch()
    print(f"[*] Создана структура в папке: '{TEST_ROOT}'")

def cleanup_test_environment():
    """Удаляет тестовые файлы и папки."""
    print("\n--- Очистка тестового окружения ---")
    if TEST_ROOT.exists():
        shutil.rmtree(TEST_ROOT)
        print(f"[*] Папка '{TEST_ROOT}' и ее содержимое удалены.")

def run_test():
    """Основная функция для запуска теста."""
    print("\n--- Тестирование инструмента: list_directory ---")

    payload = {
        "name": "list_directory",
        "arguments": {
            "directory_path": str(TEST_ROOT)
        }
    }
    
    print(f"[*] Запрашиваем содержимое папки: '{TEST_ROOT}'")

    try:
        response = requests.post(TOOL_ENDPOINT, json=payload, timeout=10)
        response.raise_for_status()
        result_data = response.json()

        print(f"[*] Получен ответ. Статус-код: {response.status_code}")

        # Проверяем, что ответ успешный и содержит ожидаемые данные
        if "result" in result_data:
            content = result_data["result"]
            expected_files = sorted(['report.txt', 'data.csv'])
            expected_dirs = ['subdir']

            if content.get("files") == expected_files and content.get("directories") == expected_dirs:
                print("\033[92m[УСПЕХ]\033[0m Инструмент вернул корректный список файлов и директорий!")
                print("--- Полученные данные ---")
                print(json.dumps(content, indent=2, ensure_ascii=False))
                print("-------------------------")
            else:
                print("\033[91m[ОШИБКА]\033[0m Содержимое ответа не соответствует ожидаемому.")
                print("Ожидалось:")
                print(json.dumps({"files": expected_files, "directories": expected_dirs}, indent=2))
                print("Получено:")
                print(json.dumps(content, indent=2, ensure_ascii=False))
        else:
            print(f"\033[91m[ОШИБКА]\033[0m Инструмент вернул ошибку: {result_data.get('error')}")

    except requests.exceptions.RequestException as e:
        print(f"\033[91m[КРИТИЧЕСКАЯ ОШИБКА]\033[0m Не удалось подключиться к серверу: {e}")

if __name__ == "__main__":
    try:
        setup_test_environment()
        run_test()
    finally:
        cleanup_test_environment()