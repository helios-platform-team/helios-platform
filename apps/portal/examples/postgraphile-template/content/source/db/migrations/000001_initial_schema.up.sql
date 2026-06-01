-- Initial schema setup for PostGraphile GraphQL API

CREATE SCHEMA IF NOT EXISTS public;

CREATE TABLE IF NOT EXISTS public.users (
  id SERIAL PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  email TEXT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.posts (
  id SERIAL PRIMARY KEY,
  author_id INTEGER NOT NULL REFERENCES public.users(id),
  title TEXT NOT NULL,
  content TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert dummy data so GraphiQL has data immediately
INSERT INTO public.users (username, email) VALUES 
  ('admin', 'admin@example.com'),
  ('guest', 'guest@example.com');

INSERT INTO public.posts (author_id, title, content) VALUES 
  (1, 'Welcome to PostGraphile', 'This is a GraphQL API generated from your database schema!');