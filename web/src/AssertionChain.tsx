import React, { useEffect, useState, useRef } from 'react';
import {graphviz} from 'd3-graphviz';
import { Graphviz } from 'graphviz-react';

interface Data {
  vis: string;
}

export const AssertionsChain = () => {
  const [dot, setDot] = useState<string>('digraph {}');

  useEffect(() => {
    const socket = new WebSocket('ws://localhost:8000/api/ws');
    socket.onmessage = (event: any) => {
      const item = JSON.parse(event.data);
      setDot(item.vis);
    };
    return () => socket.close();
  }, []);
  return (
    <section className="p-4 break-all">
        <Graphviz dot={dot} options={{'zoom': true, 'width': 800 }}></Graphviz>
    </section>
  )
};
