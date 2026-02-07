import { Component, type ReactNode } from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { Home } from './pages/Home';
import { Admin } from './pages/Admin';
import { Status } from './pages/Status';
import AvatarDebug from './pages/AvatarDebug';
import './App.css';

class ErrorBoundary extends Component<
  { children: ReactNode },
  { error: Error | null }
> {
  state = { error: null as Error | null };

  static getDerivedStateFromError(error: Error) {
    return { error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error('[ErrorBoundary]', error, info.componentStack);
  }

  render() {
    if (this.state.error) {
      return (
        <div className="min-h-screen flex items-center justify-center p-6">
          <div className="bg-matte-gray rounded-2xl p-8 max-w-md w-full text-center space-y-4">
            <h1 className="text-xl font-bold text-white">Something went wrong</h1>
            <p className="text-sm text-red-400 font-mono break-all">
              {this.state.error.message}
            </p>
            <button
              onClick={() => {
                this.setState({ error: null });
                window.location.reload();
              }}
              className="px-6 py-2 bg-yellow-neon text-indigo-deep font-semibold rounded-lg"
            >
              Reload
            </button>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}

function App() {
  return (
    <ErrorBoundary>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/admin" element={<Admin />} />
          <Route path="/status" element={<Status />} />
          <Route path="/avatar-debug" element={<AvatarDebug />} />
        </Routes>
      </BrowserRouter>
    </ErrorBoundary>
  );
}

export default App;
