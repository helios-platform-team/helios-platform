{% if values.stateManagement == 'none' or values.dataFetching == 'none' -%}
import * as React from 'react'
{%- endif %}
{% if values.stateManagement == 'zustand' -%}
import { useStore } from './store/store.js'
{%- elif values.stateManagement == 'redux' -%}
import { useSelector, useDispatch } from 'react-redux'
import { increment, decrement } from './store/store.js'
{%- endif %}

{% if values.dataFetching == 'react-query' -%}
import { useQuery } from '@tanstack/react-query'
{%- endif %}

function App() {
  {% if values.stateManagement == 'zustand' -%}
  const { count, increment, decrement } = useStore()
  {%- elif values.stateManagement == 'redux' -%}
  const count = useSelector((state) => state.counter.value)
  const dispatch = useDispatch()
  {%- else -%}
  const [count, setCount] = React.useState(0)
  {%- endif %}

  {% if values.dataFetching == 'react-query' -%}
  const { data, isLoading, error } = useQuery({
    queryKey: ['repoData'],
    queryFn: async () => {
      {% if values.backendComponent -%}
      // ponytail: Defaulting to local dev port for NestJS (3001) or postgrest (8080) to test local connection.
      const res = await fetch('http://localhost:${{ values.backendPort or 3001 }}/health')
      {%- else -%}
      const res = await fetch('https://api.github.com/repos/facebook/react')
      {%- endif %}
      if (!res.ok) {
        throw new Error('Network response was not ok')
      }
      return res.json()
    },
  })
  {%- else -%}
  const [data, setData] = React.useState(null)
  const [isLoading, setIsLoading] = React.useState(false)
  const [error, setError] = React.useState(null)

  React.useEffect(() => {
    setIsLoading(true)
    {% if values.backendComponent -%}
    // ponytail: Defaulting to local dev port for NestJS (3001) or postgrest (8080) to test local connection.
    fetch('http://localhost:${{ values.backendPort or 3001 }}/health')
      .then((res) => {
        if (!res.ok) {
          throw new Error('Network response was not ok')
        }
        return res.json()
      })
      .then((data) => {
        setData(data)
        setIsLoading(false)
      })
      .catch((err) => {
        setError(err)
        setIsLoading(false)
      })
    {%- else -%}
    fetch('https://api.github.com/repos/facebook/react')
      .then((res) => {
        if (!res.ok) {
          throw new Error('Network response was not ok')
        }
        return res.json()
      })
      .then((data) => {
        setData(data)
        setIsLoading(false)
      })
      .catch((err) => {
        setError(err)
        setIsLoading(false)
      })
    {%- endif %}
  }, [])
  {%- endif %}

  return (
    <div className="{% if values.styling == 'tailwind' %}min-h-screen bg-slate-900 text-white flex flex-col items-center justify-center p-8{% else %}app-container{% endif %}">
      <div className="{% if values.styling == 'tailwind' %}max-w-2xl w-full bg-slate-800 rounded-xl p-8 shadow-2xl border border-slate-700{% else %}app-card{% endif %}">
        <h1 className="{% if values.styling == 'tailwind' %}text-3xl font-bold mb-6 bg-gradient-to-r from-cyan-400 to-blue-500 bg-clip-text text-transparent{% else %}app-title{% endif %}">
          React App Scaffolder Template
        </h1>
        
        <div className="{% if values.styling == 'tailwind' %}mb-8 p-4 bg-slate-700/50 rounded-lg{% else %}tech-stack-panel{% endif %}">
          <h2 className="{% if values.styling == 'tailwind' %}text-lg font-semibold mb-2 text-cyan-300{% else %}section-title{% endif %}">Tech Stack</h2>
          <ul className="{% if values.styling == 'tailwind' %}grid grid-cols-2 gap-2 text-sm{% else %}tech-stack-list{% endif %}">
            <li><strong>Build Tool:</strong> ${{ values.buildTool | capitalize }}</li>
            <li><strong>Styling:</strong> ${{ values.styling | capitalize }}</li>
            <li><strong>State Management:</strong> ${{ values.stateManagement | capitalize }}</li>
            <li><strong>Data Fetching:</strong> {% if values.dataFetching == 'react-query' %}React Query{% else %}Standard Fetch{% endif %}</li>
          </ul>
        </div>

        {/* Counter Section */}
        <div className="{% if values.styling == 'tailwind' %}mb-8{% else %}counter-section{% endif %}">
          <h2 className="{% if values.styling == 'tailwind' %}text-xl font-semibold mb-4{% else %}section-title{% endif %}">Counter State Example (${{ values.stateManagement | capitalize }})</h2>
          <div className="{% if values.styling == 'tailwind' %}flex items-center gap-4{% else %}counter-controls{% endif %}">
            <button 
              onClick={() => {% if values.stateManagement == 'zustand' %}decrement(){% elif values.stateManagement == 'redux' %}dispatch(decrement()){% else %}setCount(c => c - 1){% endif %}}
              className="{% if values.styling == 'tailwind' %}px-4 py-2 bg-slate-700 hover:bg-slate-600 rounded font-bold transition-all{% else %}btn{% endif %}"
            >
              -
            </button>
            <span className="{% if values.styling == 'tailwind' %}text-2xl font-mono w-12 text-center{% else %}counter-value{% endif %}">{count}</span>
            <button 
              onClick={() => {% if values.stateManagement == 'zustand' %}increment(){% elif values.stateManagement == 'redux' %}dispatch(increment()){% else %}setCount(c => c + 1){% endif %}}
              className="{% if values.styling == 'tailwind' %}px-4 py-2 bg-cyan-600 hover:bg-cyan-500 rounded font-bold transition-all{% else %}btn btn-primary{% endif %}"
            >
              +
            </button>
          </div>
        </div>

        {/* Data Fetching Section */}
        <div className="{% if values.styling == 'tailwind' %}border-t border-slate-700 pt-6{% else %}data-section{% endif %}">
          <h2 className="{% if values.styling == 'tailwind' %}text-xl font-semibold mb-4{% else %}section-title{% endif %}">
            {% if values.backendComponent -%}
            Backend Connection Example
            {%- else -%}
            Data Fetching Example ({% if values.dataFetching == 'react-query' %}React Query{% else %}Fetch{% endif %})
            {%- endif -%}
          </h2>
          {isLoading && <p>{% if values.backendComponent %}Connecting to backend...{% else %}Loading React repo stats...{% endif %}</p>}
          {error && <p className="{% if values.styling == 'tailwind' %}text-red-400{% else %}error-msg{% endif %}">Error fetching data: {error.message || 'Unknown error'}</p>}
          {data && (
            <div className="{% if values.styling == 'tailwind' %}text-sm space-y-2{% else %}stats-grid{% endif %}">
              {% if values.backendComponent -%}
              <p><strong>Connected Backend:</strong> ${{ values.backendComponent }}</p>
              <p><strong>Health Status:</strong> {data.status}</p>
              {%- else -%}
              <p><strong>Repo Name:</strong> {data.full_name}</p>
              <p><strong>Stars:</strong> {data.stargazers_count?.toLocaleString()}</p>
              <p><strong>Forks:</strong> {data.forks_count?.toLocaleString()}</p>
              <p><strong>Open Issues:</strong> {data.open_issues_count?.toLocaleString()}</p>
              {%- endif %}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default App
