import React, { useEffect, useState } from 'react';

interface Data {
  typ: string;
  contents: string;
}

export const EventsList = () => {
  const [data, setData] = useState<Data[]>([]);

  useEffect(() => {
    const socket = new WebSocket('ws://localhost:8000/api/ws');
    socket.onmessage = (event: any) => {
      const item = JSON.parse(event.data);
      const items = [...data, item];
      console.log(items);
      setData(items);
    };
    socket.onclose = () => {
      setData([]);
    };
    return () => socket.close();
  }, data);

  let items: any = (<div>No events</div>);
  if (data.length > 0) {
    items = data.map((item: Data, idx: any) => (
        <div key={idx}>{item.typ}</div>
    ));
  }
  return (
    <section className="p-4 mt-4 mb-12 border-2 border-gray-500 flex flex-col max-w-md mx-auto">
        <div className="flex items-left font-black mb-2">
            <span className="tracking-wide text-lg text-gray-900">Events</span>
        </div>
        {items}
    </section>
  )
};
