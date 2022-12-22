import React, { useState, useEffect } from 'react';

interface FormData {
  num_validators: number;
}

export const SidebarConfig: React.FC = () => {
  const [formData, setFormData] = useState<FormData>({ num_validators: 0 });
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
      setFormData({ ...formData, num_validators: parseInt(event.target.value) });
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
      console.log(resp);
      setFormData(resp);
    } catch (e: any) {
      setError(e.message)
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <section className="p-4 flex flex-col max-w-md mx-auto">
          <div className="p-6 bg-sky-100 rounded-sm">
              <div className="flex items-left font-black mb-2">
                  <span className="tracking-wide text-lg text-gray-900">Config</span>
              </div>
              {error && <div className="text-red-500">Error: {error}</div>}
              <label className="text-sm font-medium">Num validators</label>
              <input className="mb-3 px-2 py-1.5
                  mb-3 mt-1 block w-full px-2 py-1.5 border border-gray-300 rounded-md text-sm shadow-sm placeholder-gray-400
                  focus:outline-none
                  focus:border-sky-500
                  focus:ring-1
                  focus:ring-sky-500" type="number" name="num_validators" placeholder="0" value={formData.num_validators} onChange={handleChange}/>
              <button disabled={submitting} className="bg-fuchsia-500 px-4 py-1.5 rounded-sm shadow-lg font-small text-white font-white block" type="submit">
                  { submitting ? 'Loading...' : 'Reload simulation' }
              </button>
          </div>
      </section>
    </form>
  );
};
