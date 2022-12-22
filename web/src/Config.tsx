import React, { useState, useEffect } from 'react';

interface Data {
  num_validators: number;
}

export const ConfigComponent: React.FC = () => {
  const [data, setData] = useState<string | null>(null);
  const [error, _] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      try {
        const response = await fetch('/api/config');
        const json = await response.json();
        setData(JSON.stringify(json));
      } catch (e) {
        console.error(e);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  if (loading) {
    return <div>Loading...</div>;
  }

  if (error) {
    return <div>Error: {error}</div>;
  }

  if (!data) {
    return null;
  }

  return (
    <div className="break-all">
        {data}
    </div>
  );
};
