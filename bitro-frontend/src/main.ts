import { sendToDelayedInbox } from './sendMessage';

const sendBtn = document.getElementById('sendBtn') as HTMLButtonElement;
sendBtn.onclick = async () => {
  const msg = (document.getElementById('message') as HTMLInputElement).value;

  try {
    await sendToDelayedInbox(msg);
    alert('Message sent!');
  } catch (e) {
    console.error(e);
    alert('Failed to send message.' + e);
  }
};
