import React, { useState, useEffect } from 'react';

interface FormData {
  num_validators: number;
  challenge_period_seconds: number;
  disagree_at_height: number;
  num_states: number;
}

export const SidebarConfig: React.FC = () => {
  const [formData, setFormData] = useState<FormData>({
    num_validators: 0,
    challenge_period_seconds: 0,
    disagree_at_height: 0,
    num_states: 0,
  });
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const handleValidators = (event: React.ChangeEvent<HTMLInputElement>) => {
      setFormData({
        ...formData,
        num_validators: parseInt(event.target.value),
      });
  };

  const handlePeriod = (event: React.ChangeEvent<HTMLInputElement>) => {
      setFormData({
        ...formData,
        challenge_period_seconds: parseInt(event.target.value),
      });
  };

  const handleDisagree = (event: React.ChangeEvent<HTMLInputElement>) => {
      setFormData({
        ...formData,
        disagree_at_height: parseInt(event.target.value),
      });
  };

  const handleStates = (event: React.ChangeEvent<HTMLInputElement>) => {
      setFormData({
        ...formData,
        num_states: parseInt(event.target.value),
      });
  };

  useEffect(() => {
    const fetchData = async () => {
      try {
        const response = await fetch('/api/config');
        const item: FormData = await response.json();
        setFormData(item);
      } catch (e) {
        console.error(e);
      }
    };
    fetchData();
  }, []);

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSubmitting(true);
    try {
      const response = await fetch('/api/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData),
      });
      if (!response.ok) {
        throw new Error(response.statusText);
      }
      // reset the form
      const resp: FormData = await response.json();
      setFormData(resp);
    } catch (e: any) {
      setError(e.message)
    } finally {
      setSubmitting(false);
    }
    window.location.reload();
  };

  return (
    <section className="p-4 border-2 border-gray-500 flex flex-col max-w-md mx-auto">
      <form onSubmit={handleSubmit}>
        <div className="flex items-left font-black mb-2">
            <span className="tracking-wide text-lg text-gray-900">Config</span>
        </div>
        {error && <div className="text-red-500">Error: {error}</div>}
        <label className="text-sm font-medium">Num validators</label>
        <input className="mb-3 px-2 py-1.5
            mb-3 mt-1 block w-full px-2 py-1.5 border border-gray-300 rounded-md text-sm shadow-sm placeholder-gray-400
            focus:outline-none"
            type="number"
            name="num_validators"
            placeholder="0"
            value={formData.num_validators}
            onChange={handleValidators}/>

        <label className="text-sm font-medium">Num states</label>
        <input className="mb-3 px-2 py-1.5
            mb-3 mt-1 block w-full px-2 py-1.5 border border-gray-300 rounded-md text-sm shadow-sm placeholder-gray-400
            focus:outline-none"
            type="number"
            name="num_states"
            placeholder="0"
            value={formData.num_states}
            onChange={handleStates}/>

        <label className="text-sm font-medium">Validators disagree at height</label>
        <input className="mb-3 px-2 py-1.5
            mb-3 mt-1 block w-full px-2 py-1.5 border border-gray-300 rounded-md text-sm shadow-sm placeholder-gray-400
            focus:outline-none"
            type="number"
            name="num_states"
            placeholder="0"
            value={formData.disagree_at_height}
            onChange={handleDisagree}/>

        <label className="text-sm font-medium">Challenge period (seconds)</label>
        <input className="mb-3 px-2 py-1.5
            mb-3 mt-1 block w-full px-2 py-1.5 border border-gray-300 rounded-md text-sm shadow-sm placeholder-gray-400
            focus:outline-none"
            type="number"
            name="challenge_period_seconds"
            placeholder="0"
            value={formData.challenge_period_seconds}
            onChange={handlePeriod}/>

        <button disabled={submitting} className="bg-gray-500 px-4 py-1.5 rounded-sm shadow-lg test-sm text-white font-white block" type="submit">
            { submitting ? 'Loading...' : 'Reload simulation' }
        </button>
      </form>
    </section>
  );
};
