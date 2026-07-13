export interface DataMessage {
  success: boolean;
  accessToken?: string;
  sessionToken?: string;
  user?: User;
  email?: boolean;
  error?: string;
}

interface OpenPopupProps {
  url: string;
  title: string;
  onClose?: (data: DataMessage) => void;
  width?: number;
  height?: number;
}

/**
 * Helper function that opens a popup and calls onClose when needed.
 */
const openPopup = ({
  url,
  title,
  onClose,
  width = 600,
  height = 600,
}: OpenPopupProps): Window | null => {
  const popup = window.open(
    url,
    title,
    "toolbar=no,location=no,status=no,menubar=no," +
      `scrollbars=yes,resizable=yes,width=${width},height=${height},` +
      "left=100,top=100"
  );

  // The listener that will be triggered on postMessage
  const listener = (e: MessageEvent) => {
    if (typeof e.data.success !== "undefined") {
      // Show the status for longer for better UX
      setTimeout(() => {
        if (typeof popup?.close !== "undefined") {
          popup.close();
        }

        if (typeof onClose === "function") {
          onClose(e.data);
        }
      }, 1000);

      window.removeEventListener<"message">("message", listener);
    }
  };

  window.addEventListener("message", listener);

  const timer = setInterval(function () {
    if (popup?.closed) {
      clearInterval(timer);
      onClose?.({ success: false });
    }
  }, 1000);

  return popup;
};

export default openPopup;
