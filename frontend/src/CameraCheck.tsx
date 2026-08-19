import { useEffect, useRef, useState } from 'react';
import { Camera, CheckCircle2, ShieldCheck, X, XCircle } from 'lucide-react';

type CameraState = 'idle' | 'checking' | 'ready' | 'denied' | 'unsupported';

export default function CameraCheck({ arabic, onClose, onReady }: { arabic: boolean; onClose: () => void; onReady?: () => void }) {
  const [state, setState] = useState<CameraState>('idle');
  const [message, setMessage] = useState('');
  const videoRef = useRef<HTMLVideoElement>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const t = (ar: string, en: string) => (arabic ? ar : en);

  useEffect(() => () => {
    streamRef.current?.getTracks().forEach((track) => track.stop());
  }, []);

  async function startCheck() {
    if (!navigator.mediaDevices?.getUserMedia) {
      setState('unsupported');
      setMessage(t('المتصفح لا يدعم الوصول إلى الكاميرا.', 'This browser does not support camera access.'));
      return;
    }
    setState('checking');
    setMessage(t('نطلب إذن الكاميرا الآن...', 'Requesting camera permission...'));
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: false });
      streamRef.current?.getTracks().forEach((track) => track.stop());
      streamRef.current = stream;
      if (videoRef.current) videoRef.current.srcObject = stream;
      setState('ready');
      setMessage(t('الكاميرا تعمل. سيتم تشغيل المراقبة الفعلية عند بدء الامتحان.', 'Camera is ready. Live proctoring starts when the examination begins.'));
    } catch {
      setState('denied');
      setMessage(t('لم يتم السماح بالكاميرا. فعّل الإذن من إعدادات المتصفح ثم أعد المحاولة.', 'Camera permission was not granted. Enable it in browser settings and try again.'));
    }
  }

  return <div className="camera-check-backdrop" role="dialog" aria-modal="true" aria-label={t('فحص الكاميرا', 'Camera check')}>
    <div className="camera-check-modal">
      <div className="camera-check-head">
        <div><span className="student-kicker">{t('فحص ما قبل الامتحان', 'PRE-EXAM CHECK')}</span><h2>{t('اختبار الكاميرا', 'Camera readiness check')}</h2></div>
        <button className="icon-btn" onClick={onClose} aria-label={t('إغلاق', 'Close')}><X size={18} /></button>
      </div>
      <div className="camera-check-preview">
        {state === 'ready' ? <video ref={videoRef} autoPlay muted playsInline /> : <div className="camera-check-placeholder"><Camera size={34} /><strong>{t('الكاميرا غير مفعلة بعد', 'Camera is not active yet')}</strong><span>{t('لن نطلب الإذن إلا بعد الضغط على زر الفحص.', 'Permission is requested only after you start the check.')}</span></div>}
        {state === 'ready' && <span className="camera-live-pill"><i /> {t('الكاميرا تعمل', 'Camera live')}</span>}
      </div>
      <div className="camera-check-status">
        {state === 'ready' ? <CheckCircle2 className="camera-success" size={20} /> : state === 'denied' || state === 'unsupported' ? <XCircle className="camera-error" size={20} /> : <ShieldCheck size={20} />}
        <div><strong>{state === 'ready' ? t('جاهز للمراقبة', 'Ready for monitoring') : t('المراقبة الشفافة', 'Transparent proctoring')}</strong><p>{message || t('يتم تسجيل إشارات فنية للمراجعة البشرية، ولا تُعتبر وحدها دليل مخالفة.', 'Technical signals are recorded for human review and are not proof of misconduct by themselves.')}</p></div>
      </div>
      <div className="camera-check-actions"><button className="secondary" onClick={onClose}>{t('إغلاق', 'Close')}</button>{state === 'ready' && onReady ? <button className="primary" onClick={onReady}><CheckCircle2 size={15}/>{t('متابعة إلى الامتحان', 'Continue to examination')}</button> : <button className="primary" onClick={startCheck} disabled={state === 'checking'}>{t('ابدأ فحص الكاميرا', 'Start camera check')}</button>}</div>
    </div>
  </div>;
}
