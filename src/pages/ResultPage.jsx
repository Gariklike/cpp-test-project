import ResultView from "../components/ResultView";

export default function ResultPage() {
  const answers = JSON.parse(localStorage.getItem("answers"));

  return (
    <div>
      <h1>Ваш результат</h1>
      <ResultView answers={answers} />
    </div>
  );
}
