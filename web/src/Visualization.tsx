import React, { useEffect, useState, useRef } from 'react';
import { graphviz } from '@hpcc-js/graph';

interface Data {
  typ: string;
  contents: string;
  vis: Vis;
}

interface Vis {
    assertion_chain: string;
    challenges: Challenge[];
}

interface Challenge {
    root_assertion_commit: string;
    graph: string;
}

interface SvgResp {
    svg: string;
}

export const Visualization = () => {
  const [challenges, setChallenges] = useState<string[]>([]);
  const [vis, setVis] = useState<string>();
  const [submitting, setSubmitting] = useState<boolean>(false);

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSubmitting(true);
    try {
      const response = await fetch('/api/step', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });
      if (!response.ok) {
        throw new Error(response.statusText);
      }
      setSubmitting(false);
    } catch (e: any) {
        console.error(e);
        setSubmitting(false);
    }
  };

  useEffect(() => {
    const socket = new WebSocket('ws://localhost:8000/api/ws');
    socket.onmessage = async (event: any) => {
      const data: Data = JSON.parse(event.data);
      const resp: any = await graphviz(data.vis.assertion_chain).response;
      setVis(resp.svg);

      let challengeGraphs: string[] = [];
      for (let i = 0; i < data.vis.challenges.length; i++) {
        const chal: any = await graphviz(data.vis.challenges[i].graph).response;
        challengeGraphs.push(chal.svg);
      }
      setChallenges(challengeGraphs);
    };
    return () => socket.close();
  }, [vis, challenges]);

  return (
    <div className="p-4 break-all">
        { vis &&
            <div>
                <div className="text">
                    <span className="tracking-wide text-lg text-gray-900 font-bold">Assertion Chain</span>
                </div>
                <div className="graph mt-4" dangerouslySetInnerHTML={{ __html: vis }} />
            </div>
        }
        { challenges.length > 0 &&
          challenges.map((ch: string, idx: any) => (
            <div className="mt-4">
                <div>
                    <span className="tracking-wide text-lg text-gray-900 font-bold">Challenge #{idx}</span>
                </div>
                <div>
                    <span className="tracking-wide text-md text-gray-500">
                        Challenged assertion with height 0 and commit 0x232
                    </span>
                </div>
                <div className="mt-4">
                    <form onSubmit={handleSubmit}>
                        <button
                            type="submit"
                            className="bg-gray-500 px-4 py-1.5 rounded-sm shadow-lg font-small text-white font-white block">
                            Step Through Time
                        </button>
                    </form>
                </div>
                <div className="graph mt-4" dangerouslySetInnerHTML={{ __html: ch }} />
            </div>
          ))
        }
    </div>
  )
};
