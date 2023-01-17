import React, { useEffect, useState } from 'react';

interface Data {
  typ: string;
  contents: string;
  to: string;
  from: string;
  becomes_ps: boolean;
  validator: string;
}

export const EventsList = () => {
  const [data, setData] = useState<Data[]>([]);

  useEffect(() => {
    const socket = new WebSocket('ws://localhost:8000/api/ws');
    socket.onmessage = (event: any) => {
      const item = JSON.parse(event.data);
      const items = [...data, item];
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
        <div key={idx}>
            <div className="font-bold">{item.typ}</div>
            <div className="text-gray-500 pl-2">
                { item.from && <span>from: {item.from}, </span> }
                { item.to && <span>to: {item.to}</span> }
                { item.to && <div>{ item.becomes_ps ? 'presumptive' : '' }</div> }
                { item.validator && <div>validator: {item.validator}</div> }
            </div>
        </div>
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
