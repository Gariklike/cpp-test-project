import { useState } from "react";
import { useNavigate } from "react-router-dom";
import AnswerForm from "../components/AnswerForm";

export default function LoginPage() {
  const navigate = useNavigate();
  const [login, setLogin] = useState("");
  const [password, setPassword] = useState("");

  // Временный фейковый вход по логину/паролю (для теста интерфейса)
  const handleSubmit = (e) => {
    e.preventDefault();

    const token = "fake-token";

    localStorage.setItem("token", token);
    localStorage.setItem("userId", login);

    navigate("/tests");
  };

  // Реальный вход через Google OAuth: браузер уходит на backend
  const handleGoogleLogin = () => {
    // этот URL должен быть настроен в вашем backend
    window.location.href = "http://localhost:8080/auth/google";
    // backend сам перенаправит назад на фронт, например /auth/callback?code=...
  };

  return (
    <div style={{ padding: 20 }}>
      <h1>Вход</h1>

      {/* Форма для локального фейкового входа (можно оставить как запасной вариант) */}
      <AnswerForm
        login={login}
        password={password}
        onLoginChange={setLogin}
        onPasswordChange={setPassword}
        onSubmit={handleSubmit}
      />

      <hr style={{ margin: "20px 0" }} />

      {/* Кнопка входа через Google */}
      <button onClick={handleGoogleLogin}>
        Войти через Google
      </button>
    </div>
  );
}
