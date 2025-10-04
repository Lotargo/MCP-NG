# Файл: tests/test_hybrid_search.py
# Назначение: Комплексное тестирование гибридного поиска.

import requests
import json
import sqlite3
from pathlib import Path
from sentence_transformers import SentenceTransformer

# --- Настройки ---
SERVER_URL = "http://localhost:8002"
TOOL_ENDPOINT = f"{SERVER_URL}/tools/run"
VECTOR_DB_PATH = Path("vector_db.json")
SQL_DB_PATH = Path("hybrid_db.sqlite")

# Документы, которые мы сгенерируем и загрузим в наши БД
DOCUMENTS_METADATA = [
    {"id": 1, "title": "Основы Квантов", "author": "Alice", "year": 2023, "topic": "quantum_physics"},
    {"id": 2, "title": "Введение в Python", "author": "Bob", "year": 2022, "topic": "python_programming"},
    {"id": 3, "title": "История Рима", "author": "Alice", "year": 2021, "topic": "ancient_rome"},
    {"id": 4, "title": "Продвинутый Python", "author": "Bob", "year": 2023, "topic": "python_programming"}
]

def call_tool(payload: dict) -> dict:
    """Улучшенная функция для вызова инструмента с проверкой статуса."""
    try:
        response = requests.post(TOOL_ENDPOINT, json=payload, timeout=30)
        # НОВАЯ СТРОКА: Проверяем, что сервер не вернул ошибку (4xx или 5xx)
        response.raise_for_status() 
        return response.json()
    except requests.exceptions.HTTPError as e:
        print(f"\n\033[91m[ОШИБКА HTTP]\033[0m Сервер вернул ошибку! Статус: {e.response.status_code}")
        print(f"   Ответ сервера: {e.response.text}")
        return {"critical_error": e.response.text}
    except requests.exceptions.RequestException as e:
        print(f"\n\033[91m[КРИТИЧЕСКАЯ ОШИБКА]\033[0m Не удалось подключиться: {e}")
        return {"critical_error": str(e)}

def setup_test_environment():
    print("--- Подготовка окружения для гибридного теста ---")
    
    # 1. Генерируем тексты с помощью инструмента 'text_generator'
    generated_texts = {}
    for doc in DOCUMENTS_METADATA:
        print(f"[*] Генерируем текст для '{doc['title']}'...")
        payload = {"name": "text_generator", "arguments": {"topic": doc['topic']}}
        response = call_tool(payload)
        generated_texts[doc['id']] = response['result']

    # 2. Создаем "Векторную Базу Данных" (JSON)
    print("[*] Создаем векторную базу данных (vector_db.json)...")
    model = SentenceTransformer('all-MiniLM-L6-v2')
    vector_data = []
    for doc in DOCUMENTS_METADATA:
        text = generated_texts[doc['id']]
        embedding = model.encode(text).tolist()
        vector_data.append({"id": doc['id'], "text": text, "embedding": embedding})
    with open(VECTOR_DB_PATH, 'w', encoding='utf-8') as f:
        json.dump(vector_data, f)

    # 3. Создаем СУБД (SQLite)
    print("[*] Создаем SQL базу данных (hybrid_db.sqlite)...")
    con = sqlite3.connect(SQL_DB_PATH)
    cur = con.cursor()
    cur.execute("CREATE TABLE documents(id INTEGER PRIMARY KEY, title TEXT, author TEXT, year INTEGER)")
    for doc in DOCUMENTS_METADATA:
        cur.execute("INSERT INTO documents VALUES (?, ?, ?, ?)", (doc['id'], doc['title'], doc['author'], doc['year']))
    con.commit()
    con.close()
    print("--- Окружение готово ---")

def cleanup_test_environment():
    print("\n--- Очистка ---")
    if VECTOR_DB_PATH.exists(): VECTOR_DB_PATH.unlink()
    if SQL_DB_PATH.exists(): SQL_DB_PATH.unlink()
    print("[*] Временные базы данных удалены.")

def run_test():
    print("\n--- Тестирование инструмента: hybrid_search ---")
    
    # Ищем документы про программирование (семантика) от автора Bob (точный фильтр)
    payload = {
        "name": "hybrid_search",
        "arguments": {
            "semantic_query": "лучший язык для скриптов",
            "filters": {"author": "Bob", "year": 2023}
        }
    }
    
    print("[*] Выполняем гибридный поиск...")
    result_data = call_tool(payload)
    
    if "result" in result_data:
        expected_result = [{"id": 4, "title": "Продвинутый Python", "author": "Bob", "year": 2023}]
        if result_data["result"] == expected_result:
            print("\033[92m[УСПЕХ]\033[0m Гибридный поиск вернул корректный отфильтрованный результат!")
            print(json.dumps(result_data, indent=2, ensure_ascii=False))
        else:
            print("\033[91m[ОШИБКА]\033[0m Результат не соответствует ожидаемому.")
            print("Ожидалось:", json.dumps({"result": expected_result}, indent=2, ensure_ascii=False))
            print("Получено:", json.dumps(result_data, indent=2, ensure_ascii=False))
    else:
        print(f"\033[91m[ОШИБКА]\033[0m Инструмент вернул ошибку: {result_data.get('error')}")

if __name__ == "__main__":
    try:
        setup_test_environment()
        run_test()
    finally:
        cleanup_test_environment()