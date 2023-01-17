import React, { useState } from 'react';

interface FormData {
  index: number;
}

export const ActionForm: React.FC = () => {
  const [formData, setFormData] = useState<FormData>({ index: 0 });
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setFormData({ ...formData, index: parseInt(event.target.value) });
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSubmitting(true);
    try {
      console.log('sending', formData);
      const response = await fetch('/api/assertions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData),
      });
      if (!response.ok) {
        throw new Error(response.statusText);
      }
      setFormData({ index: formData.index + 1 });
    } catch (e: any) {
      setError(e.message)
    } finally {
      setSubmitting(false);
    }
  };

  return (
      <section className="p-4 mt-4 border-2 border-gray-500 flex flex-col max-w-md mx-auto">
        <form onSubmit={handleSubmit}>
          <div className="flex items-left font-black mb-2">
              <span className="tracking-wide text-lg text-gray-900">Actions</span>
          </div>
          {error && <div className="text-red-500">Error: {error}</div>}
          <label className="text-sm font-medium">Validator #</label>
          <input className="mb-3 px-2 py-1.5
              mb-3 mt-1 block w-full px-2 py-1.5 border border-gray-300 rounded-md text-sm shadow-sm placeholder-gray-400
              focus:outline-none"
              type="number" name="index" placeholder="0" value={formData.index} onChange={handleChange}/>

          <button disabled={submitting}
                  className="bg-gray-500 px-4 py-1.5 rounded-sm shadow-lg font-small text-white font-white block"
                  type="submit">
            Submit assertion
          </button>
        </form>
      </section>
  );
};
