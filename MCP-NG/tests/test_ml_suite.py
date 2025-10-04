# Файл: tests/test_ml_suite.py

import requests
import json

SERVER_URL = "http://127.0.0.1:8002"
TOOL_ENDPOINT = f"{SERVER_URL}/tools/run"

TEST_TEXT = """
Илон Маск, глава компании SpaceX, объявил о планах по созданию 
полностью многоразовой ракетной системы Starship. 
Целью проекта является колонизация Марса и значительное 
удешевление космических полетов. Система состоит из 
корабля Starship и ускорителя Super Heavy.
"""

def run_test():
    print("--- Комплексное тестирование ML-инструментов (Summarizer & Extractor) ---")

    # --- Тест 1: Суммаризатор ---
    print("\n[1] Тестируем 'text_summarizer'...")
    summarizer_payload = {
        "name": "text_summarizer",
        "arguments": {"text": TEST_TEXT, "max_length": 60}
    }
    summarizer_response = requests.post(TOOL_ENDPOINT, json=summarizer_payload, timeout=60).json()
    if "result" in summarizer_response and summarizer_response['result']:
        print(f"\033[92m[УСПЕХ]\033[0m Получено резюме: '{summarizer_response['result']}'")
    else:
        print(f"\033[91m[ОШИБКА]\033[0m {summarizer_response}")

    # --- Тест 2: Экстрактор ---
    print("\n[2] Тестируем 'keyword_extractor'...")
    extractor_payload = {
        "name": "keyword_extractor",
        "arguments": {"text": TEST_TEXT, "max_keywords": 5}
    }
    extractor_response = requests.post(TOOL_ENDPOINT, json=extractor_payload, timeout=60).json()
    if "result" in extractor_response and extractor_response['result']:
        print(f"\033[92m[УСПЕХ]\033[0m Получены ключевые слова: {extractor_response['result']}")
    else:
        print(f"\033[91m[ОШИБКА]\033[0m {extractor_response}")

if __name__ == "__main__":
    run_test()