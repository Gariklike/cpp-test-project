import { useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { api } from "../api";

export default function AuthCallbackPage() {
  const [params] = useSearchParams();
  const navigate = useNavigate();

  useEffect(() => {
    const code = params.get("code");

    if (!code) {
      console.error("Нет кода авторизации в параметрах URL");
      return;
    }

    api
      .loginExternal(code)
      .then((data) => {
        // ожидаем, что backend вернёт token и userId
        localStorage.setItem("token", data.token);
        localStorage.setItem("userId", data.userId);

        navigate("/tests");
      })
      .catch((err) => {
        console.error("Ошибка авторизации:", err);
      });
  }, [params, navigate]);

  return <p>Авторизация… Подождите, идёт вход через Google.</p>;
}
