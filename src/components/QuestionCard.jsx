export default function QuestionCard({ question, selected, onSelect }) {
  return (
    <div style={{ border: "1px solid #ccc", padding: 16, marginBottom: 16 }}>
      <h3>{question.text}</h3>

      {question.options.map((opt) => (
        <button
          key={opt}
          onClick={() => onSelect(opt)}
          style={{
            marginRight: 8,
            marginTop: 8,
            padding: "6px 12px",
            background: selected === opt ? "#d1d1d1" : "white",
            border: "1px solid #aaa",
            cursor: "pointer",
          }}
        >
          {opt}
        </button>
      ))}
    </div>
  );
}
