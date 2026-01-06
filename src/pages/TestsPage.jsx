import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

export default function TestsPage() {
  const navigate = useNavigate();
  const [tests, setTests] = useState([]);

  useEffect(() => {
    const fakeTests = [
      { id: 1, title: "Тест по математике" },
      { id: 2, title: "Опрос по еде" },
      { id: 3, title: "Тест на общие знания" },
    ];

    // eslint-disable-next-line react-hooks/set-state-in-effect 
    setTests(fakeTests);
  }, []);

  return (
    <div style={{ padding: 20 }}>
      <h1>Доступные тесты</h1>

      {tests.length === 0 ? (
        <p>Нет доступных тестов</p>
      ) : (
        tests.map((t) => (
          <div key={t.id} style={{ marginBottom: 12 }}>
            <strong>{t.title}</strong>
            <button
              style={{ marginLeft: 10 }}
              onClick={() => navigate(`/test/${t.id}`)}
            >
              Начать
            </button>
          </div>
        ))
      )}
    </div>
  );
}
