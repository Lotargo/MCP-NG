# ==============================================================================
# Файл: tests/test_wildberries_tool.py (ИСПРАВЛЕННАЯ ВЕРСИЯ)
# Описание: Исправленный тестовый скрипт, совместимый с рабочим mcp_server.
# ==============================================================================

import requests
import time

# --- Конфигурация ---
MCP_SERVER_URL = "http://localhost:8002"
TOOL_RUN_URL = f"{MCP_SERVER_URL}/tools/run"

def print_header(title):
    print("\n" + "="*60)
    print(f" {title}")
    print("="*60)

def print_result(test_name, response):
    # Теперь мы просто печатаем статус и ответ, не пытаясь угадать "успех"
    status_code = response.status_code if hasattr(response, 'status_code') else 'N/A'
    print(f"[*] Тест: {test_name:<35} | Статус ответа: {status_code}")
    
    try:
        json_response = response.json()
        response_str = str(json_response)
        if len(response_str) > 150:
            response_str = response_str[:150] + "..."
        print(f"    Ответ: {response_str}")
        # Простая проверка на успех: код 200 и нет ключа 'error'
        if status_code == 200 and 'error' not in json_response:
             print("    Результат: ✅ УСПЕХ")
        else:
             print("    Результат: ❌ ПРОВАЛ")

    except Exception:
        print(f"    Ответ (не JSON): {response.text[:150]}...")
        print("    Результат: ❌ ПРОВАЛ")

# --- Основные тестовые вызовы ---

def run_tests():
    print_header("Тестирование инструмента Wildberries")
    print("Цель: Убедиться, что mcp_server может вызвать инструмент 'wildberries'.")

    # --- Тест 1: Простой GET-запрос (ping) ---
    test_name_1 = "Простой GET-запрос (/ping)"
    payload_1 = {
        "name": "wildberries",
        "arguments": {
            "method": "GET",
            "endpoint": "/ping",
        }
    }
    response_1 = requests.post(TOOL_RUN_URL, json=payload_1)
    print_result(test_name_1, response_1)
    
    # --- Тест 2: GET-запрос с параметрами ---
    test_name_2 = "GET с параметрами (/api/v3/orders)"
    payload_2 = {
        "name": "wildberries",
        "arguments": {
            "method": "GET",
            "endpoint": "/api/v3/orders",
            "query_params": {"limit": 2}
        }
    }
    response_2 = requests.post(TOOL_RUN_URL, json=payload_2)
    print_result(test_name_2, response_2)

    # --- Тест 3: POST-запрос с телом ---
    test_name_3 = "POST с телом (/content/v2/get/cards/list)"
    payload_3 = {
        "name": "wildberries",
        "arguments": {
            "method": "POST",
            "endpoint": "/content/v2/get/cards/list",
            "json_body": {"settings": {"cursor": {"limit": 3}}}
        }
    }
    response_3 = requests.post(TOOL_RUN_URL, json=payload_3)
    print_result(test_name_3, response_3)

if __name__ == "__main__":
    print("Убедитесь, что запущены:\n  1. wb_mock_server.py (порт 8003)\n  2. ВАШ рабочий mcp_server.py (порт 8002)")
    run_tests()
    print("\n" + "="*60)
    print("Тестирование завершено.")
    print("="*60)