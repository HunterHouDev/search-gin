import {commonAxios} from '../../boot/axios';

export const QueryAuthorList = async (data: unknown) => {
  const res = await commonAxios().post('/api/authorList', data);
  return res;
};
