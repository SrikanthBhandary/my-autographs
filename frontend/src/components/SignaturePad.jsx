import { forwardRef, useEffect, useImperativeHandle, useRef, useState } from "react";

const INKS = [
  { name: "Ink", value: "#1c1e2a" },
  { name: "Forest", value: "#2f5d50" },
  { name: "Brass", value: "#a97c3f" },
];

const SignaturePad = forwardRef(function SignaturePad({ initialImage } = {}, ref) {
  const canvasRef = useRef(null);
  const ctxRef = useRef(null);
  const isDrawing = useRef(false);
  const lastPoint = useRef(null);
  const [color, setColor] = useState(INKS[0].value);
  const [hasDrawn, setHasDrawn] = useState(false);

  // Set up the canvas backing store at device pixel ratio once on mount so
  // strokes stay crisp on retina screens. If we're remounting after the
  // user hit "Edit" and already had a signature, reload it so it isn't
  // lost — otherwise going back to preview would re-check an empty canvas.
  useEffect(() => {
    const canvas = canvasRef.current;
    const rect = canvas.getBoundingClientRect();
    const ratio = window.devicePixelRatio || 1;
    canvas.width = rect.width * ratio;
    canvas.height = rect.height * ratio;
    const ctx = canvas.getContext("2d");
    ctx.scale(ratio, ratio);
    ctx.lineCap = "round";
    ctx.lineJoin = "round";
    ctxRef.current = ctx;

    if (initialImage) {
      const img = new Image();
      img.onload = () => {
        ctx.drawImage(img, 0, 0, rect.width, rect.height);
        setHasDrawn(true);
      };
      img.src = initialImage;
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function pointFromEvent(e) {
    const rect = canvasRef.current.getBoundingClientRect();
    return { x: e.clientX - rect.left, y: e.clientY - rect.top };
  }

  function handlePointerDown(e) {
    e.preventDefault();
    try {
      canvasRef.current.setPointerCapture(e.pointerId);
    } catch {
      // Some browsers/pointer types don't support capture — safe to ignore.
    }
    isDrawing.current = true;
    const point = pointFromEvent(e);
    lastPoint.current = point;

    // Draw an immediate dot so a single tap (no drag) still leaves a mark
    // and counts as "signed" — a quick tap alone would otherwise draw nothing.
    const ctx = ctxRef.current;
    ctx.fillStyle = color;
    ctx.beginPath();
    ctx.arc(point.x, point.y, 1.25, 0, Math.PI * 2);
    ctx.fill();
    setHasDrawn(true);
  }

  function handlePointerMove(e) {
    if (!isDrawing.current) return;
    e.preventDefault();
    const ctx = ctxRef.current;
    const point = pointFromEvent(e);
    ctx.strokeStyle = color;
    ctx.lineWidth = 2.5;
    ctx.beginPath();
    ctx.moveTo(lastPoint.current.x, lastPoint.current.y);
    ctx.lineTo(point.x, point.y);
    ctx.stroke();
    lastPoint.current = point;
  }

  function handlePointerUp() {
    isDrawing.current = false;
  }

  function clear() {
    const canvas = canvasRef.current;
    ctxRef.current.clearRect(0, 0, canvas.width, canvas.height);
    setHasDrawn(false);
  }

  useImperativeHandle(ref, () => ({
    isEmpty: () => !hasDrawn,
    clear,
    toDataURL: () => canvasRef.current.toDataURL("image/png"),
    toBlob: () =>
      new Promise((resolve) => {
        canvasRef.current.toBlob(resolve, "image/png");
      }),
  }));

  return (
    <div className="signature-pad">
      <div className="signature-canvas-wrap">
        {!hasDrawn && (
          <div className="signature-guideline">
            <span>Sign here</span>
          </div>
        )}
        <canvas
          ref={canvasRef}
          className="signature-canvas"
          onPointerDown={handlePointerDown}
          onPointerMove={handlePointerMove}
          onPointerUp={handlePointerUp}
          onPointerLeave={handlePointerUp}
          onPointerCancel={handlePointerUp}
        />
      </div>
      <div className="signature-pad-controls">
        <div className="signature-inks">
          {INKS.map((ink) => (
            <button
              key={ink.value}
              type="button"
              className={`ink-swatch${color === ink.value ? " selected" : ""}`}
              style={{ background: ink.value }}
              onClick={() => setColor(ink.value)}
              aria-label={`${ink.name} ink`}
              title={ink.name}
            />
          ))}
        </div>
        <button type="button" className="ghost" onClick={clear} disabled={!hasDrawn}>
          Clear
        </button>
      </div>
    </div>
  );
});

export default SignaturePad;
