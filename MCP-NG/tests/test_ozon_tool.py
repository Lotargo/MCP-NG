# ==============================================================================
# Файл: tests/test_ozon_tool.py (ИСПРАВЛЕННАЯ ВЕРСИЯ)
# Описание: Тестирует РЕАЛЬНЫЙ инструмент ozon.py через mcp_server,
#           который обращается к ozon_mock_server.py.
# ==============================================================================

import requests

MCP_SERVER_URL = "http://localhost:8002"
TOOL_RUN_URL = f"{MCP_SERVER_URL}/tools/run"

def print_result(test_name, response):
    status_code = response.status_code
    print(f"[*] Тест: {test_name:<35} | Статус ответа: {status_code}")
    json_response = response.json()
    print(f"    Ответ: {str(json_response)[:150]}...")
    if status_code == 200 and 'error' not in json_response and json_response.get('result', {}).get('mocked'):
        print("    Результат: ✅ УСПЕХ")
    else:
        print("    Результат: ❌ ПРОВАЛ")

def run_tests():
    print("\n" + "="*60 + "\n Тестирование инструмента Ozon\n" + "="*60)
    
    # --- Тест 1: Получение списка товаров ---
    payload1 = {"name": "ozon", "arguments": {"endpoint": "/v2/product/list", "payload": {"page_size": 3}}}
    response1 = requests.post(TOOL_RUN_URL, json=payload1)
    print_result("Получение списка товаров", response1)

    # --- Тест 2: Получение списка заказов FBS ---
    payload2 = {"name": "ozon", "arguments": {"endpoint": "/v3/posting/fbs/list", "payload": {"limit": 5}}}
    response2 = requests.post(TOOL_RUN_URL, json=payload2)
    print_result("Получение заказов FBS", response2)

if __name__ == "__main__":
    print("Убедитесь, что запущены:\n  1. ozon_mock_server.py (порт 8004)\n  2. mcp_server.py (порт 8002)")
    run_tests()
    print("\nТестирование завершено.")