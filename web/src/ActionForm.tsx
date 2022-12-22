import React, { useState } from 'react';

interface FormData {
  index: number;
}

export const ActionForm: React.FC = () => {
  const [formData, setFormData] = useState<FormData>({ index: 0 });
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setFormData({ ...formData, [event.target.name]: event.target.value });
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSubmitting(true);
    try {
      const response = await fetch('/api/assertions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData),
      });
      if (!response.ok) {
        throw new Error(response.statusText);
      }
      // reset the form
      setFormData({ index: 0 });
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
              <div className="flex items-left font-black mt-2 mb-2">
                  <span className="tracking-wide text-lg text-gray-900">Validator actions</span>
              </div>
              {error && <div className="text-red-500">Error: {error}</div>}
              <label className="text-sm font-medium">Validator #</label>
              <input className="mb-3 px-2 py-1.5
                  mb-3 mt-1 block w-full px-2 py-1.5 border border-gray-300 rounded-md text-sm shadow-sm placeholder-gray-400
                  focus:outline-none
                  focus:border-sky-500
                  focus:ring-1
                  focus:ring-sky-500" type="text" name="index" placeholder="0" value={formData.index} onChange={handleChange}/>
              <button disabled={submitting} className="bg-fuchsia-500 px-4 py-1.5 rounded-sm shadow-lg font-small text-white font-white block" type="submit">
                Submit assertion
              </button>
          </div>
      </section>
    </form>
  );
};
