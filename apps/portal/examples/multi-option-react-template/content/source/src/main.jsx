import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.jsx'
import './index.css'

{% if values.dataFetching == 'react-query' -%}
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
const queryClient = new QueryClient()
{%- endif %}

{% if values.stateManagement == 'redux' -%}
import { Provider } from 'react-redux'
import { store } from './store/store.js'
{%- endif %}

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    {% if values.dataFetching == 'react-query' -%}
    <QueryClientProvider client={queryClient}>
    {%- endif %}
      {% if values.stateManagement == 'redux' -%}
      <Provider store={store}>
      {%- endif %}
        <App />
      {% if values.stateManagement == 'redux' -%}
      </Provider>
      {%- endif %}
    {% if values.dataFetching == 'react-query' -%}
    </QueryClientProvider>
    {%- endif %}
  </React.StrictMode>,
)
