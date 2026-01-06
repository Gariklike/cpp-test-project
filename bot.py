import telebot
import requests
from config import BOT_TOKEN

bot = telebot.TeleBot(BOT_TOKEN)

# Состояния: user_id -> {current, answers, questions, score}
user_states = {}

def get_questions():
    # Ожидается формат:
    # [
    #   {"id": 1, "text": "...", "options": ["A", "B", "C"], "correct": "B"}
    #   ИЛИ {"id": 2, "text": "...", "options": ["A","B","C"], "correct_index": 1}
    # ]
    resp = requests.get("http://127.0.0.1:5000/questions")
    resp.raise_for_status()
    return resp.json()

def send_results(user_id, answers):
    # На сервер уходит список выбранных индексов или текстов
    resp = requests.post("http://127.0.0.1:5000/results",
                         json={"user_id": user_id, "answers": answers})
    resp.raise_for_status()
    return resp.json()

@bot.message_handler(commands=['start'])
def start(message):
    bot.send_message(message.chat.id, "Привет! Напиши /test чтобы пройти тест.")

@bot.message_handler(commands=['test'])
def start_test(message):
    questions = get_questions()
    user_states[message.chat.id] = {
        "current": 0,
        "answers": [],
        "questions": questions,
        "score": 0
    }
    send_question(message.chat.id)

def send_question(chat_id):
    state = user_states.get(chat_id)
    if not state:
        return
    q = state["questions"][state["current"]]
    markup = telebot.types.InlineKeyboardMarkup()
    # Используем индекс как callback_data для надёжности
    for idx, option in enumerate(q["options"]):
        markup.add(telebot.types.InlineKeyboardButton(text=option, callback_data=str(idx)))
    bot.send_message(chat_id, q["text"], reply_markup=markup)

@bot.callback_query_handler(func=lambda call: True)
def handle_answer(call):
    chat_id = call.message.chat.id
    state = user_states.get(chat_id)
    if not state:
        return

    q = state["questions"][state["current"]]

    # Индекс выбранного варианта
    try:
        selected_idx = int(call.data)
    except ValueError:
        # На случай старых кнопок с текстом как callback_data
        selected_idx = q["options"].index(call.data) if call.data in q["options"] else -1

    # Определяем правильный ответ (поддержка двух форматов)
    is_correct = False
    if "correct_index" in q:
        is_correct = (selected_idx == q["correct_index"])
    elif "correct" in q:
        correct_text = q["correct"]
        selected_text = q["options"][selected_idx] if 0 <= selected_idx < len(q["options"]) else None
        is_correct = (selected_text == correct_text)
    else:
        # Если сервер не вернул признак правильного ответа — считаем неизвестным
        bot.answer_callback_query(call.id, "Нет данных о правильном ответе на сервере.")
        return

    # Обратная связь пользователю
    feedback = "Верно ✅" if is_correct else "Неверно ❌"
    bot.answer_callback_query(call.id, feedback)

    # Обновляем состояние
    state["answers"].append(selected_idx)
    if is_correct:
        state["score"] += 1
    state["current"] += 1

    # Переходим к следующему вопросу или завершаем
    if state["current"] < len(state["questions"]):
        send_question(chat_id)
    else:
        total = len(state["questions"])
        score = state["score"]
        bot.send_message(chat_id, f"Тест завершён. Результат: {score}/{total}")

        user_states.pop(chat_id, None)

print("Бот запущен...")
bot.infinity_polling()
