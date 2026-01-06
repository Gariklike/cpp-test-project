export default function AnswerForm({ login, password, onLoginChange, onPasswordChange, onSubmit }) {
  return (
    <form onSubmit={onSubmit}>
      <input
        placeholder="Логин"
        value={login}
        onChange={(e) => onLoginChange(e.target.value)}
      />
      <input
        type="password"
        placeholder="Пароль"
        value={password}
        onChange={(e) => onPasswordChange(e.target.value)}
      />
      <button type="submit">Войти</button>
    </form>
  );
}
