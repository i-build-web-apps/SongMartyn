import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { Home } from './pages/Home';
import { Admin } from './pages/Admin';
import { Status } from './pages/Status';
import AvatarDebug from './pages/AvatarDebug';
import './App.css';

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/admin" element={<Admin />} />
        <Route path="/status" element={<Status />} />
        <Route path="/avatar-debug" element={<AvatarDebug />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
