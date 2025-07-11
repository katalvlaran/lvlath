package main

// city codes (92)
const (
	Alchevsk              = "Alchevsk"
	Bakhmut               = "Bakhmut"
	Berdiansk             = "Berdiansk"
	Berdychiv             = "Berdychiv"
	Bila_Tserkva          = "Bila_Tserkva"
	Bilhorod_Dnistrovskyi = "Bilhorod_Dnistrovskyi"
	Boryspil              = "Boryspil"
	Brovary               = "Brovary"
	Bucha                 = "Bucha"
	Cherkasy              = "Cherkasy"
	Chernihiv             = "Chernihiv"
	Chernivtsi            = "Chernivtsi"
	Chernobyl             = "Chernobyl"
	Chornomorsk           = "Chornomorsk"
	Chortkiv              = "Chortkiv"
	Chuhuiv               = "Chuhuiv"
	Dnipro                = "Dnipro"
	Dobropillia           = "Dobropillia"
	Donetsk               = "Donetsk"
	Drohobych             = "Drohobych"
	Dubno                 = "Dubno"
	Dzhankoi              = "Dzhankoi"
	Enerhodar             = "Enerhodar"
	Feodosiia             = "Feodosiia"
	Henichesk             = "Henichesk"
	Horishni_Plavni       = "Horishni-Plavni"
	Horlivka              = "Horlivka"
	Ivano_Frankivsk       = "Ivano-Frankivsk"
	Izmail                = "Izmail"
	Izyum                 = "Izyum"
	Kamianets_Podilskyi   = "Kamianets-Podilskyi"
	Kamianske             = "Kamianske"
	Kaniv                 = "Kaniv"
	Kharkiv               = "Kharkiv"
	Kherson               = "Kherson"
	Khmelnytskyi          = "Khmelnytskyi"
	Kolomyia              = "Kolomyia"
	Kovel                 = "Kovel"
	Kramatorsk            = "Kramatorsk"
	Krasnyi_Luch          = "Krasnyi-Luch"
	Kremenchuk            = "Kremenchuk"
	Kropyvnytskyi         = "Kropyvnytskyi"
	Kryvyi_Rih            = "Kryvyi-Rih"
	Kupiansk              = "Kupiansk"
	Kyiv                  = "Kyiv"
	Ladyzhyn              = "Ladyzhyn"
	Luhansk               = "Luhansk"
	Lutsk                 = "Lutsk"
	Lviv                  = "Lviv"
	Lysychansk            = "Lysychansk"
	Mariupol              = "Mariupol"
	Melitopol             = "Melitopol"
	Mukachevo             = "Mukachevo"
	Mykolaiv              = "Mykolaiv"
	Myrhorod              = "Myrhorod"
	Nikopol               = "Nikopol"
	Nizhyn                = "Nizhyn"
	Nova_Kakhovka         = "Nova-Kakhovka"
	Novhorod_Siverskyi    = "Novhorod-Siverskyi"
	Novomoskovsk          = "Novomoskovsk"
	Novovolynsk           = "Novovolynsk"
	Odesa                 = "Odesa"
	Oleksandriia          = "Oleksandriia"
	Ovruch                = "Ovruch"
	Pavlohrad             = "Pavlohrad"
	Pervomaisk            = "Pervomaisk"
	Pokrovsk              = "Pokrovsk"
	Poltava               = "Poltava"
	Rivne                 = "Rivne"
	Romny                 = "Romny"
	Rubizhne              = "Rubizhne"
	Sarny                 = "Sarny"
	Sevastopol            = "Sevastopol"
	Shcherbyny            = "Shcherbyny"
	Simferopol            = "Simferopol"
	Sloviansk             = "Sloviansk"
	Starobilsk            = "Starobilsk"
	Sumy                  = "Sumy"
	Ternopil              = "Ternopil"
	Tokmak                = "Tokmak"
	Trostianets           = "Trostianets"
	Truskavets            = "Truskavets"
	Uman                  = "Uman"
	Uzhhorod              = "Uzhhorod"
	Vinnytsia             = "Vinnytsia"
	Volodymyr             = "Volodymyr"
	Yalta                 = "Yalta"
	Yuzhnoukrainsk        = "Yuzhnoukrainsk"
	Zaporizhzhia          = "Zaporizhzhia"
	Zhmerynka             = "Zhmerynka"
	Zhytomyr              = "Zhytomyr"
	Zolotonosha           = "Zolotonosha"
)

// ~~~~~~~~~~~~~~~~~~~~~ Transportation Network ~~~~~~~~~~~~~~~~~~~~~
// contains (334+200+27=)561 segments of Ukrainian ways(Road|Rail|Air)
var (
	// RoadNetwork contains (337) predefined segments of Ukrainian road network.
	RoadNetwork = []WaySegment{
		{From: Alchevsk, To: Luhansk, KM: 45.9},
		{From: Alchevsk, To: Rubizhne, KM: 78.0},
		{From: Alchevsk, To: Starobilsk, KM: 94.5},
		{From: Bakhmut, To: Kramatorsk, KM: 30.6},
		{From: Bakhmut, To: Lysychansk, KM: 87.1},
		{From: Bakhmut, To: Rubizhne, KM: 92.4},
		{From: Bakhmut, To: Sloviansk, KM: 43.2},
		{From: Berdiansk, To: Donetsk, KM: 183.2},
		{From: Berdiansk, To: Horlivka, KM: 229.5},
		{From: Berdiansk, To: Krasnyi_Luch, KM: 254.2},
		{From: Berdiansk, To: Luhansk, KM: 317.9},
		{From: Berdiansk, To: Mariupol, KM: 79.3},
		{From: Berdiansk, To: Melitopol, KM: 124.7},
		{From: Berdiansk, To: Sevastopol, KM: 399.9},
		{From: Berdiansk, To: Shcherbyny, KM: 307.2},
		{From: Berdiansk, To: Simferopol, KM: 332.2},
		{From: Berdiansk, To: Yalta, KM: 372.1},
		{From: Berdiansk, To: Zaporizhzhia, KM: 198.7},
		{From: Berdychiv, To: Khmelnytskyi, KM: 146.0},
		{From: Berdychiv, To: Kyiv, KM: 188.4},
		{From: Berdychiv, To: Rivne, KM: 203.0},
		{From: Berdychiv, To: Vinnytsia, KM: 140.2},
		{From: Berdychiv, To: Zhytomyr, KM: 43.7},
		{From: Bila_Tserkva, To: Brovary, KM: 104.9},
		{From: Bila_Tserkva, To: Bucha, KM: 93.9},
		{From: Bila_Tserkva, To: Cherkasy, KM: 167.1},
		{From: Bila_Tserkva, To: Chernihiv, KM: 235.8},
		{From: Bila_Tserkva, To: Chernobyl, KM: 187.8},
		{From: Bila_Tserkva, To: Khmelnytskyi, KM: 264.1},
		{From: Bila_Tserkva, To: Kyiv, KM: 88.3},
		{From: Bila_Tserkva, To: Lviv, KM: 502.5},
		{From: Bila_Tserkva, To: Uman, KM: 135.9},
		{From: Bilhorod_Dnistrovskyi, To: Chornomorsk, KM: 56.9},
		{From: Bilhorod_Dnistrovskyi, To: Izmail, KM: 116.1},
		{From: Bilhorod_Dnistrovskyi, To: Mykolaiv, KM: 189.7},
		{From: Bilhorod_Dnistrovskyi, To: Odesa, KM: 78.8},
		{From: Bilhorod_Dnistrovskyi, To: Yuzhnoukrainsk, KM: 241.0},
		{From: Boryspil, To: Brovary, KM: 42.5},
		{From: Boryspil, To: Cherkasy, KM: 145.2},
		{From: Boryspil, To: Chernihiv, KM: 160.9},
		{From: Boryspil, To: Kyiv, KM: 34.6},
		{From: Boryspil, To: Poltava, KM: 265.4},
		{From: Brovary, To: Bucha, KM: 46.8},
		{From: Brovary, To: Cherkasy, KM: 171.9},
		{From: Brovary, To: Chernihiv, KM: 132.7},
		{From: Brovary, To: Chernobyl, KM: 108.0},
		{From: Brovary, To: Kyiv, KM: 22.7},
		{From: Brovary, To: Poltava, KM: 330.8},
		{From: Brovary, To: Sumy, KM: 328.9},
		{From: Brovary, To: Zhytomyr, KM: 176.6},
		{From: Bucha, To: Cherkasy, KM: 206.7},
		{From: Bucha, To: Chernihiv, KM: 149.9},
		{From: Bucha, To: Chernobyl, KM: 94.0},
		{From: Bucha, To: Kyiv, KM: 27.8},
		{From: Bucha, To: Vinnytsia, KM: 219.9},
		{From: Bucha, To: Zhytomyr, KM: 131.9},
		{From: Cherkasy, To: Chernobyl, KM: 278.1},
		{From: Cherkasy, To: Kharkiv, KM: 351.9},
		{From: Cherkasy, To: Kremenchuk, KM: 123.2},
		{From: Cherkasy, To: Kyiv, KM: 180.3},
		{From: Cherkasy, To: Pervomaisk, KM: 206.0},
		{From: Cherkasy, To: Poltava, KM: 207.7},
		{From: Cherkasy, To: Sumy, KM: 292.1},
		{From: Cherkasy, To: Uman, KM: 177.7},
		{From: Chernihiv, To: Chernobyl, KM: 89.8},
		{From: Chernihiv, To: Kyiv, KM: 147.5},
		{From: Chernihiv, To: Sumy, KM: 291.2},
		{From: Chernihiv, To: Zhytomyr, KM: 265.3},
		{From: Chernivtsi, To: Ivano_Frankivsk, KM: 131.2},
		{From: Chernivtsi, To: Kamianets_Podilskyi, KM: 74.6},
		{From: Chernivtsi, To: Khmelnytskyi, KM: 170.0},
		{From: Chernivtsi, To: Lviv, KM: 254.3},
		{From: Chernivtsi, To: Rivne, KM: 298.8},
		{From: Chernivtsi, To: Ternopil, KM: 163.9},
		{From: Chernivtsi, To: Uzhhorod, KM: 312.2},
		{From: Chernivtsi, To: Vinnytsia, KM: 246.0},
		{From: Chernobyl, To: Kyiv, KM: 108.4},
		{From: Chernobyl, To: Zhytomyr, KM: 181.8},
		{From: Chornomorsk, To: Odesa, KM: 29.3},
		{From: Chortkiv, To: Ivano_Frankivsk, KM: 119.8},
		{From: Chortkiv, To: Kolomyia, KM: 105.2},
		{From: Chortkiv, To: Ternopil, KM: 69.8},
		{From: Chuhuiv, To: Izyum, KM: 79.2},
		{From: Chuhuiv, To: Kharkiv, KM: 38.7},
		{From: Chuhuiv, To: Kupiansk, KM: 67.5},
		{From: Dnipro, To: Donetsk, KM: 241.6},
		{From: Dnipro, To: Kharkiv, KM: 219.1},
		{From: Dnipro, To: Kremenchuk, KM: 157.5},
		{From: Dnipro, To: Kryvyi_Rih, KM: 158.0},
		{From: Dnipro, To: Luhansk, KM: 361.2},
		{From: Dnipro, To: Melitopol, KM: 208.5},
		{From: Dnipro, To: Nikopol, KM: 126.6},
		{From: Dnipro, To: Poltava, KM: 149.6},
		{From: Dnipro, To: Sumy, KM: 313.1},
		{From: Dnipro, To: Zaporizhzhia, KM: 80.4},
		{From: Dobropillia, To: Bakhmut, KM: 78.6},
		{From: Dobropillia, To: Kramatorsk, KM: 103.4},
		{From: Donetsk, To: Horlivka, KM: 46.3},
		{From: Donetsk, To: Kharkiv, KM: 285.2},
		{From: Donetsk, To: Krasnyi_Luch, KM: 98.1},
		{From: Donetsk, To: Luhansk, KM: 146.7},
		{From: Donetsk, To: Mariupol, KM: 119.4},
		{From: Donetsk, To: Melitopol, KM: 258.0},
		{From: Donetsk, To: Nikopol, KM: 297.2},
		{From: Donetsk, To: Poltava, KM: 339.7},
		{From: Donetsk, To: Shcherbyny, KM: 160.2},
		{From: Donetsk, To: Zaporizhzhia, KM: 229.3},
		{From: Drohobych, To: Ivano_Frankivsk, KM: 113.0},
		{From: Drohobych, To: Lviv, KM: 77.0},
		{From: Dubno, To: Lutsk, KM: 85.9},
		{From: Dubno, To: Lviv, KM: 135.1},
		{From: Dubno, To: Rivne, KM: 44.4},
		{From: Dubno, To: Sarny, KM: 109.6},
		{From: Dzhankoi, To: Feodosiia, KM: 123.9},
		{From: Dzhankoi, To: Melitopol, KM: 183.2},
		{From: Dzhankoi, To: Simferopol, KM: 93.8},
		{From: Enerhodar, To: Nova_Kakhovka, KM: 104.4},
		{From: Enerhodar, To: Tokmak, KM: 96.0},
		{From: Enerhodar, To: Zaporizhzhia, KM: 126.3},
		{From: Feodosiia, To: Melitopol, KM: 326.7},
		{From: Feodosiia, To: Simferopol, KM: 106.6},
		{From: Henichesk, To: Kherson, KM: 161.5},
		{From: Henichesk, To: Melitopol, KM: 86.4},
		{From: Henichesk, To: Nova_Kakhovka, KM: 196.8},
		{From: Henichesk, To: Tokmak, KM: 168.2},
		{From: Horishni_Plavni, To: Dnipro, KM: 151.6},
		{From: Horishni_Plavni, To: Kremenchuk, KM: 21.4},
		{From: Horishni_Plavni, To: Poltava, KM: 104.8},
		{From: Horlivka, To: Kharkiv, KM: 261.3},
		{From: Horlivka, To: Krasnyi_Luch, KM: 80.6},
		{From: Horlivka, To: Luhansk, KM: 110.4},
		{From: Horlivka, To: Mariupol, KM: 164.3},
		{From: Horlivka, To: Poltava, KM: 335.1},
		{From: Horlivka, To: Shcherbyny, KM: 138.4},
		{From: Ivano_Frankivsk, To: Kamianets_Podilskyi, KM: 160.7},
		{From: Ivano_Frankivsk, To: Khmelnytskyi, KM: 201.1},
		{From: Ivano_Frankivsk, To: Lutsk, KM: 238.8},
		{From: Ivano_Frankivsk, To: Lviv, KM: 130.3},
		{From: Ivano_Frankivsk, To: Rivne, KM: 251.5},
		{From: Ivano_Frankivsk, To: Ternopil, KM: 109.3},
		{From: Ivano_Frankivsk, To: Uzhhorod, KM: 207.9},
		{From: Izyum, To: Kharkiv, KM: 121.0},
		{From: Izyum, To: Kupiansk, KM: 54.9},
		{From: Izyum, To: Sloviansk, KM: 63.7},
		{From: Kamianets_Podilskyi, To: Khmelnytskyi, KM: 100.6},
		{From: Kamianets_Podilskyi, To: Lviv, KM: 259.4},
		{From: Kamianets_Podilskyi, To: Rivne, KM: 249.0},
		{From: Kamianets_Podilskyi, To: Ternopil, KM: 138.7},
		{From: Kamianets_Podilskyi, To: Uzhhorod, KM: 363.1},
		{From: Kamianets_Podilskyi, To: Vinnytsia, KM: 173.9},
		{From: Kamianske, To: Dnipro, KM: 24.1},
		{From: Kamianske, To: Kremenchuk, KM: 118.8},
		{From: Kamianske, To: Kryvyi_Rih, KM: 132.6},
		{From: Kaniv, To: Cherkasy, KM: 56.3},
		{From: Kaniv, To: Kyiv, KM: 125.4},
		{From: Kaniv, To: Zolotonosha, KM: 58.7},
		{From: Kharkiv, To: Krasnyi_Luch, KM: 330.9},
		{From: Kharkiv, To: Kremenchuk, KM: 261.0},
		{From: Kharkiv, To: Luhansk, KM: 314.4},
		{From: Kharkiv, To: Nikopol, KM: 345.7},
		{From: Kharkiv, To: Poltava, KM: 148.0},
		{From: Kharkiv, To: Shcherbyny, KM: 370.2},
		{From: Kharkiv, To: Sumy, KM: 165.0},
		{From: Kharkiv, To: Zaporizhzhia, KM: 290.4},
		{From: Kherson, To: Kryvyi_Rih, KM: 174.3},
		{From: Kherson, To: Melitopol, KM: 242.3},
		{From: Kherson, To: Mykolaiv, KM: 68.3},
		{From: Kherson, To: Odesa, KM: 167.9},
		{From: Kherson, To: Sevastopol, KM: 272.7},
		{From: Kherson, To: Simferopol, KM: 254.7},
		{From: Kherson, To: Yalta, KM: 308.3},
		{From: Kherson, To: Yuzhnoukrainsk, KM: 163.8},
		{From: Khmelnytskyi, To: Lutsk, KM: 217.3},
		{From: Khmelnytskyi, To: Lviv, KM: 250.9},
		{From: Khmelnytskyi, To: Rivne, KM: 164.3},
		{From: Khmelnytskyi, To: Ternopil, KM: 117.1},
		{From: Khmelnytskyi, To: Uman, KM: 284.2},
		{From: Khmelnytskyi, To: Uzhhorod, KM: 407.5},
		{From: Khmelnytskyi, To: Vinnytsia, KM: 126.7},
		{From: Khmelnytskyi, To: Zhytomyr, KM: 173.7},
		{From: Kolomyia, To: Chernivtsi, KM: 88.2},
		{From: Kolomyia, To: Ivano_Frankivsk, KM: 56.3},
		{From: Kolomyia, To: Lviv, KM: 175.0},
		{From: Kolomyia, To: Mukachevo, KM: 228.0},
		{From: Kolomyia, To: Uzhhorod, KM: 240.0},
		{From: Kovel, To: Rivne, KM: 121.0},
		{From: Krasnyi_Luch, To: Luhansk, KM: 66.9},
		{From: Krasnyi_Luch, To: Mariupol, KM: 176.7},
		{From: Krasnyi_Luch, To: Shcherbyny, KM: 62.1},
		{From: Krasnyi_Luch, To: Zaporizhzhia, KM: 327.3},
		{From: Kremenchuk, To: Kryvyi_Rih, KM: 148.9},
		{From: Kremenchuk, To: Kyiv, KM: 297.1},
		{From: Kremenchuk, To: Mykolaiv, KM: 294.8},
		{From: Kremenchuk, To: Poltava, KM: 114.8},
		{From: Kremenchuk, To: Sumy, KM: 260.3},
		{From: Kremenchuk, To: Uman, KM: 272.1},
		{From: Kremenchuk, To: Zaporizhzhia, KM: 214.9},
		{From: Kryvyi_Rih, To: Mykolaiv, KM: 169.8},
		{From: Kryvyi_Rih, To: Nikopol, KM: 97.5},
		{From: Kryvyi_Rih, To: Odesa, KM: 295.0},
		{From: Kryvyi_Rih, To: Pervomaisk, KM: 218.0},
		{From: Kryvyi_Rih, To: Poltava, KM: 235.8},
		{From: Kryvyi_Rih, To: Uman, KM: 289.8},
		{From: Kryvyi_Rih, To: Yuzhnoukrainsk, KM: 184.5},
		{From: Kryvyi_Rih, To: Zaporizhzhia, KM: 150.3},
		{From: Kyiv, To: Lutsk, KM: 423.5},
		{From: Kyiv, To: Lviv, KM: 537.7},
		{From: Kyiv, To: Odesa, KM: 507.7},
		{From: Kyiv, To: Poltava, KM: 348.8},
		{From: Kyiv, To: Rivne, KM: 347.8},
		{From: Kyiv, To: Sumy, KM: 351.3},
		{From: Kyiv, To: Uman, KM: 219.1},
		{From: Kyiv, To: Vinnytsia, KM: 229.4},
		{From: Kyiv, To: Zhytomyr, KM: 154.2},
		{From: Ladyzhyn, To: Kropyvnytskyi, KM: 153.0},
		{From: Ladyzhyn, To: Uman, KM: 166.4},
		{From: Ladyzhyn, To: Vinnytsia, KM: 64.3},
		{From: Luhansk, To: Mariupol, KM: 241.8},
		{From: Luhansk, To: Shcherbyny, KM: 59.9},
		{From: Lutsk, To: Lviv, KM: 157.1},
		{From: Lutsk, To: Rivne, KM: 76.8},
		{From: Lutsk, To: Ternopil, KM: 154.3},
		{From: Lutsk, To: Uzhhorod, KM: 370.2},
		{From: Lutsk, To: Zhytomyr, KM: 278.3},
		{From: Lviv, To: Rivne, KM: 207.3},
		{From: Lviv, To: Ternopil, KM: 134.5},
		{From: Lviv, To: Uzhhorod, KM: 213.1},
		{From: Lviv, To: Vinnytsia, KM: 377.5},
		{From: Lviv, To: Zhytomyr, KM: 383.7},
		{From: Mariupol, To: Melitopol, KM: 192.9},
		{From: Mariupol, To: Shcherbyny, KM: 228.0},
		{From: Melitopol, To: Mykolaiv, KM: 295.1},
		{From: Melitopol, To: Sevastopol, KM: 329.4},
		{From: Melitopol, To: Simferopol, KM: 267.2},
		{From: Melitopol, To: Yalta, KM: 319.1},
		{From: Melitopol, To: Zaporizhzhia, KM: 128.1},
		{From: Mykolaiv, To: Nikopol, KM: 222.6},
		{From: Mykolaiv, To: Odesa, KM: 128.1},
		{From: Mykolaiv, To: Pervomaisk, KM: 168.7},
		{From: Mykolaiv, To: Sevastopol, KM: 331.0},
		{From: Mykolaiv, To: Simferopol, KM: 319.5},
		{From: Mykolaiv, To: Uman, KM: 273.0},
		{From: Mykolaiv, To: Yalta, KM: 371.2},
		{From: Mykolaiv, To: Yuzhnoukrainsk, KM: 96.9},
		{From: Mykolaiv, To: Zaporizhzhia, KM: 293.6},
		{From: Myrhorod, To: Kyiv, KM: 207.9},
		{From: Myrhorod, To: Poltava, KM: 101.6},
		{From: Myrhorod, To: Sumy, KM: 149.0},
		{From: Myrhorod, To: Vinnytsia, KM: 302.0},
		{From: Nikopol, To: Odesa, KM: 349.9},
		{From: Nikopol, To: Poltava, KM: 258.2},
		{From: Nikopol, To: Sevastopol, KM: 385.9},
		{From: Nikopol, To: Simferopol, KM: 336.0},
		{From: Nikopol, To: Yalta, KM: 393.5},
		{From: Nikopol, To: Zaporizhzhia, KM: 71.7},
		{From: Nizhyn, To: Chernihiv, KM: 85.0},
		{From: Nizhyn, To: Kyiv, KM: 129.0},
		{From: Nova_Kakhovka, To: Kherson, KM: 71.0},
		{From: Nova_Kakhovka, To: Kryvyi_Rih, KM: 173.0},
		{From: Nova_Kakhovka, To: Melitopol, KM: 219.0},
		{From: Nova_Kakhovka, To: Oleksandriia, KM: 233.9},
		{From: Novhorod_Siverskyi, To: Chernihiv, KM: 170.2},
		{From: Novhorod_Siverskyi, To: Romny, KM: 177.4},
		{From: Novhorod_Siverskyi, To: Sumy, KM: 132.5},
		{From: Novomoskovsk, To: Dnipro, KM: 27.3},
		{From: Novomoskovsk, To: Kamianske, KM: 69.5},
		{From: Novomoskovsk, To: Pavlohrad, KM: 66.0},
		{From: Novovolynsk, To: Kovel, KM: 71.5},
		{From: Novovolynsk, To: Lutsk, KM: 79.8},
		{From: Novovolynsk, To: Volodymyr, KM: 27.9},
		{From: Odesa, To: Pervomaisk, KM: 200.0},
		{From: Odesa, To: Sevastopol, KM: 346.2},
		{From: Odesa, To: Simferopol, KM: 359.7},
		{From: Odesa, To: Uman, KM: 293.0},
		{From: Odesa, To: Vinnytsia, KM: 400.8},
		{From: Odesa, To: Yalta, KM: 399.4},
		{From: Odesa, To: Yuzhnoukrainsk, KM: 148.2},
		{From: Oleksandriia, To: Dnipro, KM: 159.0},
		{From: Oleksandriia, To: Kremenchuk, KM: 93.7},
		{From: Oleksandriia, To: Kropyvnytskyi, KM: 33.4},
		{From: Oleksandriia, To: Kryvyi_Rih, KM: 113.0},
		{From: Oleksandriia, To: Poltava, KM: 170.5},
		{From: Oleksandriia, To: Zaporizhzhia, KM: 284.1},
		{From: Ovruch, To: Chernihiv, KM: 211.3},
		{From: Ovruch, To: Sarny, KM: 112.7},
		{From: Ovruch, To: Zhytomyr, KM: 154.6},
		{From: Pavlohrad, To: Dnipro, KM: 86.0},
		{From: Pavlohrad, To: Kharkiv, KM: 195.0},
		{From: Pavlohrad, To: Zaporizhzhia, KM: 174.0},
		{From: Pervomaisk, To: Uman, KM: 104.7},
		{From: Pervomaisk, To: Vinnytsia, KM: 251.3},
		{From: Pervomaisk, To: Yuzhnoukrainsk, KM: 72.0},
		{From: Pokrovsk, To: Dobropillia, KM: 30.8},
		{From: Pokrovsk, To: Kramatorsk, KM: 92.1},
		{From: Pokrovsk, To: Zaporizhzhia, KM: 241.7},
		{From: Poltava, To: Sumy, KM: 169.9},
		{From: Poltava, To: Zaporizhzhia, KM: 229.2},
		{From: Rivne, To: Ternopil, KM: 146.6},
		{From: Rivne, To: Uzhhorod, KM: 416.0},
		{From: Rivne, To: Vinnytsia, KM: 255.2},
		{From: Rivne, To: Zhytomyr, KM: 201.5},
		{From: Romny, To: Chernihiv, KM: 164.0},
		{From: Romny, To: Kyiv, KM: 195.0},
		{From: Romny, To: Poltava, KM: 181.0},
		{From: Rubizhne, To: Kupiansk, KM: 97.0},
		{From: Rubizhne, To: Lysychansk, KM: 12.0},
		{From: Rubizhne, To: Starobilsk, KM: 42.0},
		{From: Sarny, To: Kovel, KM: 122.1},
		{From: Sarny, To: Lutsk, KM: 113.0},
		{From: Sarny, To: Rivne, KM: 60.0},
		{From: Sarny, To: Zhytomyr, KM: 196.8},
		{From: Sevastopol, To: Simferopol, KM: 67.9},
		{From: Sevastopol, To: Yalta, KM: 60.2},
		{From: Shcherbyny, To: Zaporizhzhia, KM: 389.3},
		{From: Simferopol, To: Yalta, KM: 58.3},
		{From: Starobilsk, To: Kupiansk, KM: 106.0},
		{From: Ternopil, To: Uzhhorod, KM: 301.5},
		{From: Ternopil, To: Vinnytsia, KM: 243.7},
		{From: Tokmak, To: Melitopol, KM: 92.0},
		{From: Tokmak, To: Zaporizhzhia, KM: 127.0},
		{From: Trostianets, To: Poltava, KM: 179.0},
		{From: Trostianets, To: Sumy, KM: 58.0},
		{From: Truskavets, To: Ivano_Frankivsk, KM: 141.0},
		{From: Truskavets, To: Lviv, KM: 97.0},
		{From: Uman, To: Vinnytsia, KM: 158.7},
		{From: Uman, To: Yuzhnoukrainsk, KM: 176.6},
		{From: Vinnytsia, To: Yuzhnoukrainsk, KM: 320.3},
		{From: Vinnytsia, To: Zhytomyr, KM: 131.5},
		{From: Volodymyr, To: Dubno, KM: 132.0},
		{From: Volodymyr, To: Lutsk, KM: 74.0},
		{From: Zhmerynka, To: Khmelnytskyi, KM: 127.0},
		{From: Zhmerynka, To: Vinnytsia, KM: 35.0},
		{From: Zolotonosha, To: Cherkasy, KM: 34.2},
		{From: Zolotonosha, To: Uman, KM: 158.4},
	}

	// RailwayNetwork contains (200) predefined segments of Ukrainian railway network.
	RailwayNetwork = []WaySegment{
		{From: Alchevsk, To: Lysychansk, KM: 84.0},
		{From: Alchevsk, To: Rubizhne, KM: 78.0},
		{From: Alchevsk, To: Starobilsk, KM: 95.0},
		{From: Berdiansk, To: Mariupol, KM: 152.0},
		{From: Berdiansk, To: Melitopol, KM: 128.0},
		{From: Berdiansk, To: Tokmak, KM: 104.0},
		{From: Berdychiv, To: Rivne, KM: 203.0},
		{From: Berdychiv, To: Vinnytsia, KM: 140.2},
		{From: Berdychiv, To: Zhytomyr, KM: 43.7},
		{From: Bila_Tserkva, To: Vinnytsia, KM: 232.0},
		{From: Bila_Tserkva, To: Zhytomyr, KM: 177.0},
		{From: Bucha, To: Kyiv, KM: 27.8},
		{From: Bucha, To: Zhytomyr, KM: 132.0},
		{From: Cherkasy, To: Chernihiv, KM: 270.0},
		{From: Cherkasy, To: Kropyvnytskyi, KM: 120.2},
		{From: Cherkasy, To: Kyiv, KM: 180.3},
		{From: Cherkasy, To: Mykolaiv, KM: 315.8},
		{From: Cherkasy, To: Odesa, KM: 395.6},
		{From: Cherkasy, To: Poltava, KM: 207.7},
		{From: Cherkasy, To: Sumy, KM: 292.1},
		{From: Chernihiv, To: Brovary, KM: 132.7},
		{From: Chernihiv, To: Kyiv, KM: 147.5},
		{From: Chernihiv, To: Poltava, KM: 360.4},
		{From: Chernihiv, To: Sumy, KM: 291.2},
		{From: Chernihiv, To: Zhytomyr, KM: 265.3},
		{From: Chernivtsi, To: Ivano_Frankivsk, KM: 131.2},
		{From: Chernivtsi, To: Khmelnytskyi, KM: 170.0},
		{From: Chernivtsi, To: Kolomyia, KM: 88.2},
		{From: Chernivtsi, To: Ternopil, KM: 163.9},
		{From: Chernivtsi, To: Uzhhorod, KM: 312.2},
		{From: Chernivtsi, To: Vinnytsia, KM: 246.0},
		{From: Chernobyl, To: Kyiv, KM: 158.0},
		{From: Chernobyl, To: Ovruch, KM: 98.0},
		{From: Chernobyl, To: Zhytomyr, KM: 182.0},
		{From: Chornomorsk, To: Bilhorod_Dnistrovskyi, KM: 56.9},
		{From: Chortkiv, To: Chernivtsi, KM: 164.0},
		{From: Chortkiv, To: Kamianets_Podilskyi, KM: 140.0},
		{From: Chortkiv, To: Ternopil, KM: 69.8},
		{From: Chuhuiv, To: Izyum, KM: 79.2},
		{From: Chuhuiv, To: Kharkiv, KM: 38.7},
		{From: Chuhuiv, To: Kupiansk, KM: 67.5},
		{From: Dnipro, To: Donetsk, KM: 241.6},
		{From: Dnipro, To: Kharkiv, KM: 219.1},
		{From: Dnipro, To: Kherson, KM: 312.4},
		{From: Dnipro, To: Kropyvnytskyi, KM: 236.3},
		{From: Dnipro, To: Luhansk, KM: 361.2},
		{From: Dnipro, To: Poltava, KM: 149.6},
		{From: Dnipro, To: Zaporizhzhia, KM: 80.4},
		{From: Dobropillia, To: Donetsk, KM: 72.4},
		{From: Dobropillia, To: Kramatorsk, KM: 90.0},
		{From: Dobropillia, To: Pokrovsk, KM: 30.8},
		{From: Donetsk, To: Kharkiv, KM: 285.2},
		{From: Donetsk, To: Luhansk, KM: 146.7},
		{From: Donetsk, To: Poltava, KM: 339.7},
		{From: Donetsk, To: Zaporizhzhia, KM: 229.3},
		{From: Dzhankoi, To: Feodosiia, KM: 123.9},
		{From: Dzhankoi, To: Melitopol, KM: 183.2},
		{From: Dzhankoi, To: Simferopol, KM: 93.8},
		{From: Feodosiia, To: Simferopol, KM: 106.6},
		{From: Horishni_Plavni, To: Dnipro, KM: 151.6},
		{From: Horishni_Plavni, To: Kremenchuk, KM: 21.4},
		{From: Horishni_Plavni, To: Poltava, KM: 104.8},
		{From: Horlivka, To: Donetsk, KM: 46.3},
		{From: Horlivka, To: Kramatorsk, KM: 112.0},
		{From: Horlivka, To: Krasnyi_Luch, KM: 80.6},
		{From: Horlivka, To: Luhansk, KM: 110.0},
		{From: Ivano_Frankivsk, To: Drohobych, KM: 113.0},
		{From: Ivano_Frankivsk, To: Khmelnytskyi, KM: 201.1},
		{From: Ivano_Frankivsk, To: Kolomyia, KM: 56.3},
		{From: Ivano_Frankivsk, To: Lviv, KM: 130.3},
		{From: Ivano_Frankivsk, To: Ternopil, KM: 109.3},
		{From: Ivano_Frankivsk, To: Uzhhorod, KM: 207.9},
		{From: Izmail, To: Bilhorod_Dnistrovskyi, KM: 116.1},
		{From: Izyum, To: Kharkiv, KM: 121.0},
		{From: Izyum, To: Kupiansk, KM: 54.9},
		{From: Izyum, To: Sloviansk, KM: 63.7},
		{From: Kamianets_Podilskyi, To: Khmelnytskyi, KM: 100.6},
		{From: Kamianets_Podilskyi, To: Ternopil, KM: 139.0},
		{From: Kamianets_Podilskyi, To: Zhmerynka, KM: 90.0},
		{From: Kamianske, To: Dnipro, KM: 24.1},
		{From: Kamianske, To: Kremenchuk, KM: 118.8},
		{From: Kamianske, To: Kryvyi_Rih, KM: 132.6},
		{From: Kaniv, To: Cherkasy, KM: 56.3},
		{From: Kaniv, To: Kyiv, KM: 125.4},
		{From: Kaniv, To: Zolotonosha, KM: 48.7},
		{From: Kharkiv, To: Luhansk, KM: 314.4},
		{From: Kharkiv, To: Poltava, KM: 148.0},
		{From: Kharkiv, To: Sumy, KM: 165.0},
		{From: Kharkiv, To: Zaporizhzhia, KM: 290.4},
		{From: Kherson, To: Henichesk, KM: 161.5},
		{From: Kherson, To: Kropyvnytskyi, KM: 240.0},
		{From: Kherson, To: Mykolaiv, KM: 68.3},
		{From: Kherson, To: Nova_Kakhovka, KM: 71.0},
		{From: Kherson, To: Odesa, KM: 167.9},
		{From: Kherson, To: Zaporizhzhia, KM: 266.3},
		{From: Khmelnytskyi, To: Kyiv, KM: 318.9},
		{From: Khmelnytskyi, To: Lutsk, KM: 217.3},
		{From: Khmelnytskyi, To: Rivne, KM: 164.3},
		{From: Khmelnytskyi, To: Ternopil, KM: 117.1},
		{From: Khmelnytskyi, To: Vinnytsia, KM: 126.7},
		{From: Khmelnytskyi, To: Zhytomyr, KM: 173.7},
		{From: Kovel, To: Lutsk, KM: 72.0},
		{From: Kovel, To: Sarny, KM: 122.1},
		{From: Krasnyi_Luch, To: Donetsk, KM: 98.1},
		{From: Krasnyi_Luch, To: Horlivka, KM: 80.6},
		{From: Krasnyi_Luch, To: Luhansk, KM: 66.9},
		{From: Kropyvnytskyi, To: Kyiv, KM: 286.6},
		{From: Kropyvnytskyi, To: Mykolaiv, KM: 198.0},
		{From: Kropyvnytskyi, To: Odesa, KM: 291.6},
		{From: Kropyvnytskyi, To: Oleksandriia, KM: 33.4},
		{From: Kropyvnytskyi, To: Zaporizhzhia, KM: 260.3},
		{From: Kryvyi_Rih, To: Oleksandriia, KM: 113.0},
		{From: Kyiv, To: Bila_Tserkva, KM: 88.3},
		{From: Kyiv, To: Boryspil, KM: 34.6},
		{From: Kyiv, To: Brovary, KM: 22.7},
		{From: Kyiv, To: Kharkiv, KM: 489.0},
		{From: Kyiv, To: Lviv, KM: 569.0},
		{From: Kyiv, To: Poltava, KM: 348.8},
		{From: Kyiv, To: Rivne, KM: 347.8},
		{From: Kyiv, To: Sumy, KM: 351.3},
		{From: Kyiv, To: Vinnytsia, KM: 229.4},
		{From: Kyiv, To: Zhytomyr, KM: 154.2},
		{From: Luhansk, To: Poltava, KM: 418.9},
		{From: Luhansk, To: Zaporizhzhia, KM: 367.5},
		{From: Lutsk, To: Dubno, KM: 85.9},
		{From: Lutsk, To: Lviv, KM: 157.1},
		{From: Lutsk, To: Rivne, KM: 76.8},
		{From: Lutsk, To: Ternopil, KM: 154.3},
		{From: Lutsk, To: Uzhhorod, KM: 370.2},
		{From: Lviv, To: Drohobych, KM: 77.0},
		{From: Lviv, To: Dubno, KM: 135.1},
		{From: Lviv, To: Rivne, KM: 207.3},
		{From: Lviv, To: Ternopil, KM: 134.5},
		{From: Lviv, To: Truskavets, KM: 97.0},
		{From: Lviv, To: Uzhhorod, KM: 213.1},
		{From: Mariupol, To: Berdiansk, KM: 165.0},
		{From: Mariupol, To: Donetsk, KM: 119.4},
		{From: Mariupol, To: Melitopol, KM: 196.0},
		{From: Melitopol, To: Henichesk, KM: 86.4},
		{From: Mukachevo, To: Kolomyia, KM: 228.0},
		{From: Mykolaiv, To: Odesa, KM: 128.1},
		{From: Mykolaiv, To: Zaporizhzhia, KM: 293.6},
		{From: Myrhorod, To: Kyiv, KM: 207.9},
		{From: Myrhorod, To: Poltava, KM: 101.6},
		{From: Myrhorod, To: Sumy, KM: 149.0},
		{From: Nizhyn, To: Chernihiv, KM: 85.0},
		{From: Nizhyn, To: Kyiv, KM: 129.0},
		{From: Nizhyn, To: Romny, KM: 150.0},
		{From: Nova_Kakhovka, To: Enerhodar, KM: 104.4},
		{From: Novhorod_Siverskyi, To: Chernihiv, KM: 170.2},
		{From: Novhorod_Siverskyi, To: Romny, KM: 177.4},
		{From: Novhorod_Siverskyi, To: Sumy, KM: 132.5},
		{From: Novomoskovsk, To: Dnipro, KM: 27.3},
		{From: Novomoskovsk, To: Kamianske, KM: 69.5},
		{From: Novomoskovsk, To: Pavlohrad, KM: 66.0},
		{From: Novovolynsk, To: Kovel, KM: 71.5},
		{From: Novovolynsk, To: Lutsk, KM: 79.8},
		{From: Novovolynsk, To: Volodymyr, KM: 27.9},
		{From: Odesa, To: Bilhorod_Dnistrovskyi, KM: 78.8},
		{From: Odesa, To: Vinnytsia, KM: 400.8},
		{From: Ovruch, To: Chernihiv, KM: 211.3},
		{From: Ovruch, To: Sarny, KM: 112.7},
		{From: Ovruch, To: Zhytomyr, KM: 154.6},
		{From: Pervomaisk, To: Kropyvnytskyi, KM: 120.0},
		{From: Pervomaisk, To: Odesa, KM: 200.0},
		{From: Pervomaisk, To: Vinnytsia, KM: 251.0},
		{From: Pokrovsk, To: Dobropillia, KM: 30.8},
		{From: Pokrovsk, To: Kramatorsk, KM: 92.1},
		{From: Pokrovsk, To: Zaporizhzhia, KM: 241.7},
		{From: Poltava, To: Boryspil, KM: 265.4},
		{From: Poltava, To: Sumy, KM: 169.9},
		{From: Poltava, To: Zaporizhzhia, KM: 229.2},
		{From: Rivne, To: Dubno, KM: 44.4},
		{From: Rivne, To: Sarny, KM: 60.0},
		{From: Rivne, To: Ternopil, KM: 146.6},
		{From: Rivne, To: Zhytomyr, KM: 201.5},
		{From: Simferopol, To: Sevastopol, KM: 67.9},
		{From: Sloviansk, To: Bakhmut, KM: 43.2},
		{From: Sloviansk, To: Kramatorsk, KM: 12.0},
		{From: Ternopil, To: Uzhhorod, KM: 301.5},
		{From: Ternopil, To: Vinnytsia, KM: 243.7},
		{From: Tokmak, To: Enerhodar, KM: 96.0},
		{From: Trostianets, To: Poltava, KM: 179.0},
		{From: Trostianets, To: Romny, KM: 121.0},
		{From: Trostianets, To: Sumy, KM: 58.0},
		{From: Vinnytsia, To: Ladyzhyn, KM: 64.3},
		{From: Vinnytsia, To: Zhytomyr, KM: 131.5},
		{From: Yuzhnoukrainsk, To: Ladyzhyn, KM: 90.0},
		{From: Yuzhnoukrainsk, To: Mykolaiv, KM: 97.0},
		{From: Yuzhnoukrainsk, To: Vinnytsia, KM: 320.0},
		{From: Zaporizhzhia, To: Enerhodar, KM: 126.3},
		{From: Zaporizhzhia, To: Melitopol, KM: 124.0},
		{From: Zaporizhzhia, To: Nikopol, KM: 71.7},
		{From: Zaporizhzhia, To: Tokmak, KM: 127.0},
		{From: Zhmerynka, To: Kamianets_Podilskyi, KM: 174.0},
		{From: Zhmerynka, To: Khmelnytskyi, KM: 127.0},
		{From: Zhmerynka, To: Vinnytsia, KM: 35.0},
		{From: Zolotonosha, To: Cherkasy, KM: 34.2},
		{From: Zolotonosha, To: Kaniv, KM: 71.9},
		{From: Zolotonosha, To: Uman, KM: 158.4},
	}

	// AirNetwork contains (27) predefined segments of Ukrainian air network.
	AirNetwork = []WaySegment{
		{From: Boryspil, To: Dnipro, KM: 391.0},
		{From: Boryspil, To: Kharkiv, KM: 409.0},
		{From: Boryspil, To: Lviv, KM: 468.0},
		{From: Boryspil, To: Odesa, KM: 441.0},
		{From: Boryspil, To: Sevastopol, KM: 720.0},
		{From: Boryspil, To: Simferopol, KM: 679.0},
		{From: Dnipro, To: Simferopol, KM: 470.0},
		{From: Kharkiv, To: Sevastopol, KM: 450.0},
		{From: Kharkiv, To: Simferopol, KM: 392.0},
		{From: Kyiv, To: Dnipro, KM: 391.0},
		{From: Kyiv, To: Kharkiv, KM: 409.0},
		{From: Kyiv, To: Lviv, KM: 470.0},
		{From: Kyiv, To: Odesa, KM: 441.0},
		{From: Lviv, To: Dnipro, KM: 861.0},
		{From: Lviv, To: Kharkiv, KM: 896.0},
		{From: Lviv, To: Odesa, KM: 628.0},
		{From: Lviv, To: Sevastopol, KM: 1092.0},
		{From: Lviv, To: Simferopol, KM: 1050.0},
		{From: Odesa, To: Dnipro, KM: 388.0},
		{From: Odesa, To: Kharkiv, KM: 558.0},
		{From: Odesa, To: Sevastopol, KM: 470.0},
		{From: Odesa, To: Simferopol, KM: 429.0},
		{From: Simferopol, To: Odesa, KM: 429.0},
	}
)
