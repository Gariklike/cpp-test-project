import { Routes, Route, Navigate } from "react-router-dom";

import LoginPage from "./pages/LoginPage";
import TestsPage from "./pages/TestsPage";
import TestPage from "./pages/TestPage";
import ResultPage from "./pages/ResultPage";
import AuthCallbackPage from "./pages/AuthCallbackPage";

export default function AppRouter() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/tests" element={<TestsPage />} />
      <Route path="/test/:id" element={<TestPage />} />
      <Route path="/result/:id" element={<ResultPage />} />

      {/* обработка возврата после Google OAuth */}
      <Route path="/auth/callback" element={<AuthCallbackPage />} />

      {/* если путь не найден — отправляем на логин */}
      <Route path="*" element={<Navigate to="/login" />} />
    </Routes>
  );
}
