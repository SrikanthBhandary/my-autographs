import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "../api/client";

const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080/api";
const WS_URL = API_URL.replace(/^http/, "ws").replace(/\/api\/?$/, "/api/ws");

// Manages the whole "export as PDF" flow: creates the job, listens for the
// worker's completion notification over WebSocket, and falls back to
// polling if the socket disconnects or the notification is somehow missed.
export function usePdfExport() {
  const [status, setStatus] = useState("idle"); // idle | pending | completed | failed
  const [fileUrl, setFileUrl] = useState(null);
  const [error, setError] = useState(null);
  const jobIdRef = useRef(null);
  const socketRef = useRef(null);
  const pollRef = useRef(null);

  function stopPolling() {
    if (pollRef.current) {
      clearInterval(pollRef.current);
      pollRef.current = null;
    }
  }

  const connectSocket = useCallback(() => {
    const token = localStorage.getItem("token");
    if (!token || socketRef.current) return;

    const socket = new WebSocket(`${WS_URL}?token=${encodeURIComponent(token)}`);
    socket.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.job_id !== jobIdRef.current) return; // not our current job
        if (msg.status === "completed") {
          setStatus("completed");
          setFileUrl(msg.file_url);
          stopPolling();
        } else if (msg.status === "failed") {
          setStatus("failed");
          setError(msg.error || "Export failed");
          stopPolling();
        }
      } catch {
        // ignore malformed messages
      }
    };
    socket.onclose = () => {
      socketRef.current = null;
    };
    socketRef.current = socket;
  }, []);

  // Fallback: poll every 4s in case the WS message never arrives (socket
  // dropped, reconnect race, etc). Cheap and self-cancelling once resolved.
  function startPolling(jobId) {
    stopPolling();
    pollRef.current = setInterval(async () => {
      try {
        const job = await api.getExportJob(jobId);
        if (job.status === "completed") {
          setStatus("completed");
          setFileUrl(job.file_url);
          stopPolling();
        } else if (job.status === "failed") {
          setStatus("failed");
          setError(job.error || "Export failed");
          stopPolling();
        }
      } catch {
        // transient errors are fine, just try again next tick
      }
    }, 4000);
  }

  const startExport = useCallback(
    async (categoryId) => {
      setStatus("pending");
      setError(null);
      setFileUrl(null);
      connectSocket();
      try {
        const job = await api.exportPDF(categoryId);
        jobIdRef.current = job.id;
        startPolling(job.id);
      } catch (err) {
        setStatus("failed");
        setError(err.message);
      }
    },
    [connectSocket]
  );

  useEffect(() => {
    return () => {
      stopPolling();
      socketRef.current?.close();
    };
  }, []);

  return { status, fileUrl, error, startExport };
}
