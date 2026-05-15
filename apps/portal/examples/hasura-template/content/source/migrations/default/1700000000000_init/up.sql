-- Create a test users table
CREATE TABLE public.users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert some dummy data
INSERT INTO public.users (name, email) VALUES 
('Alice', 'alice@example.com'),
('Bob', 'bob@example.com');