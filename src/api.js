const API_URL = "http://localhost:8080"; // потом замените на реальный адрес

// Универсальная функция запроса
async function request(path, method = "GET", body = null, token = null) {
  const headers = {
    "Content-Type": "application/json",
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const options = {
    method,
    headers,
  };

  if (body) {
    options.body = JSON.stringify(body);
  }

  const response = await fetch(API_URL + path, options);

  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(error.message || "Ошибка запроса");
  }

  // если backend ничего не вернул в теле
  return response.json().catch(() => ({}));
}

// API методы
export const api = {
  // Авторизация через внешний сервис (Google OAuth, код приходит в query-параметре)
  loginExternal(code) {
    return request(`/auth/login?code=${code}`, "GET");
  },

  getTests(token) {
    return request(`/tests`, "GET", null, token);
  },

  getTestById(id, token) {
    return request(`/tests/${id}`, "GET", null, token);
  },

  sendAnswers(testId, answers, token) {
    return request(`/tests/${testId}/answers`, "POST", { answers }, token);
  },

  getResult(attemptId, token) {
    return request(`/results/${attemptId}`, "GET", null, token);
  },
};
