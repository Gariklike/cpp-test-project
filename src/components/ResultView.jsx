export default function ResultView({ answers }) {
  if (!answers) {
    return <p>Нет данных</p>;
  }

  return (
    <ul>
      {Object.entries(answers).map(([id, answer]) => (
        <li key={id}>
          Вопрос {id}: {answer}
        </li>
      ))}
    </ul>
  );
}
