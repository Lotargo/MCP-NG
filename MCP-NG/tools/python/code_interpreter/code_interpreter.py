# Файл: tools/code_interpreter.py

import io
import sys
from contextlib import redirect_stdout

def code_interpreter(code: str) -> dict:
    """
    Выполняет предоставленный код на Python и возвращает результат (вывод print).
    КРАЙНЕ ОПАСНО! Этот инструмент выполняет произвольный код.
    Используйте для математических вычислений, обработки текста и алгоритмических задач.
    
    :param code: Строка, содержащая корректный код на Python.
    :return: Словарь с выводом кода или сообщением об ошибке.
    """
    # Создаем "ловушку" для стандартного вывода (stdout)
    output_buffer = io.StringIO()
    
    try:
        # Перенаправляем все вызовы print() в наш буфер
        with redirect_stdout(output_buffer):
            # ВНИМАНИЕ: exec() - это самая опасная часть. Она выполняет код как есть.
            exec(code)
            
        # Получаем все, что было "напечатано"
        captured_output = output_buffer.getvalue()
        
        return {"result": captured_output}

    except Exception as e:
        # Если в коде произошла ошибка, возвращаем ее текст
        return {"error": f"Ошибка выполнения кода: {type(e).__name__}: {e}"}