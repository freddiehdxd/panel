import { Routes, Route, Navigate } from 'react-router-dom';
import { useEffect, useState } from 'react';
import { api } from '@/lib/api';

import Login from '@/pages/Login';
import Dashboard from '@/pages/Dashboard';
import Apps from '@/pages/Apps';
import Domains from '@/pages/Domains';
import SSL from '@/pages/SSL';
import Databases from '@/pages/Databases';
import Redis from '@/pages/Redis';
import Files from '@/pages/Files';
import Logs from '@/pages/Logs';
import NotFound from '@/pages/NotFound';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const [authed, setAuthed] = useState<boolean | null>(null);

  useEffect(() => {
    api.get<{ username: string }>('/auth/me').then((res) => {
      setAuthed(res.success);
    });
  }, []);

  if (authed === null) {
    // Loading state
    return (
      <div className="flex min-h-screen items-center justify-center bg-[#080810]">
        <span className="h-5 w-5 rounded-full border-2 border-violet-500/30 border-t-violet-500 animate-spin" />
      </div>
    );
  }

  if (!authed) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/" element={<Navigate to="/dashboard" replace />} />
      <Route path="/dashboard" element={<ProtectedRoute><Dashboard /></ProtectedRoute>} />
      <Route path="/apps" element={<ProtectedRoute><Apps /></ProtectedRoute>} />
      <Route path="/domains" element={<ProtectedRoute><Domains /></ProtectedRoute>} />
      <Route path="/ssl" element={<ProtectedRoute><SSL /></ProtectedRoute>} />
      <Route path="/databases" element={<ProtectedRoute><Databases /></ProtectedRoute>} />
      <Route path="/redis" element={<ProtectedRoute><Redis /></ProtectedRoute>} />
      <Route path="/files" element={<ProtectedRoute><Files /></ProtectedRoute>} />
      <Route path="/logs" element={<ProtectedRoute><Logs /></ProtectedRoute>} />
      <Route path="*" element={<NotFound />} />
    </Routes>
  );
}
