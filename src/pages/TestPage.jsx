import { useState } from "react";
import { useNavigate } from "react-router-dom";
import QuestionCard from "../components/QuestionCard";

export default function TestPage() {
  const navigate = useNavigate();

  // Временно используем тестовые вопросы
  const questions = [
    { id: 1, text: "Ваш любимый цвет?", options: ["Красный", "Синий", "Зелёный"] },
    { id: 2, text: "Любимая еда?", options: ["Пицца", "Суши", "Бургер"] },
  ];

  const [answers, setAnswers] = useState({});

  const handleAnswer = (id, option) => {
    setAnswers({ ...answers, [id]: option });
  };

  const handleFinish = () => {
    localStorage.setItem("answers", JSON.stringify(answers));
    navigate("/result");
  };

  return (
    <div style={{ padding: 20 }}>
      <h1>Тест</h1>

      {questions.map((q) => (
        <QuestionCard
          key={q.id}
          question={q}
          selected={answers[q.id]}
          onSelect={(opt) => handleAnswer(q.id, opt)}
        />
      ))}

      <button
        onClick={handleFinish}
        style={{
          padding: "10px 20px",
          fontSize: "16px",
          cursor: "pointer",
          marginTop: "20px",
        }}
      >
        Завершить
      </button>
    </div>
  );
}
