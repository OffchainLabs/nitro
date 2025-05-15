import { sendToDelayedInbox } from './sendMessage';

const sendBtn = document.getElementById('sendBtn') as HTMLButtonElement;
sendBtn.onclick = async () => {
  const l3 = (document.getElementById('l3Address') as HTMLInputElement).value;
  const msg = (document.getElementById('message') as HTMLInputElement).value;

  try {
    await sendToDelayedInbox(l3, msg);
    alert('Message sent!');
  } catch (e) {
    console.error(e);
    alert('Failed to send message.' + e);
  }
};
